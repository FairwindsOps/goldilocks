// Copyright 2019 FairwindsOps Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package vpa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/samber/lo"
	"k8s.io/client-go/util/retry"
	"golang.org/x/time/rate"

	autoscaling "k8s.io/api/autoscaling/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"

	controllerLog "github.com/fairwindsops/controller-utils/pkg/log"
	"github.com/fairwindsops/goldilocks/pkg/kube"
	"github.com/fairwindsops/goldilocks/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/textlogger"
)

// Reconciler checks if VPA objects should be created or deleted
type Reconciler struct {
	KubeClient            *kube.ClientInstance
	VPAClient             *kube.VPAClientInstance
	DynamicClient         *kube.DynamicClientInstance
	ControllerUtilsClient *kube.ControllerUtilsClientInstance
	OnByDefault           bool
	DryRun                bool
	IncludeNamespaces     []string
	ExcludeNamespaces     []string
	IgnoreControllerKind  []string
	operationMutex        sync.Mutex     // Protects bulk operations
	operationSemaphore    chan struct{}  // Limits concurrent VPA operations
	activeOperationsWG    sync.WaitGroup // Tracks active VPA operations
	reconciliationMutex   sync.RWMutex   // Coordinates between reconciliation and new events
	rateLimiter           *rate.Limiter  // Limits control plane API calls to 10/second
}

type Controller struct {
	APIVersion   string
	Kind         string
	Name         string
	Unstructured *unstructured.Unstructured
}

var singleton *Reconciler
var controllerUtilsLogr = textlogger.NewLogger(textlogger.NewConfig())

// getControlPlaneRateLimit returns the rate limit for control plane API calls from environment variable
// Defaults to 10 calls per second if not set or invalid
func getControlPlaneRateLimit() rate.Limit {
	if rateLimitStr := os.Getenv("GOLDILOCKS_CONTROL_PLANE_RATE_LIMIT"); rateLimitStr != "" {
		if rateLimit, err := strconv.ParseFloat(rateLimitStr, 64); err == nil && rateLimit > 0 {
			klog.Infof("🏃 Control plane rate limit set to %.1f calls/second via GOLDILOCKS_CONTROL_PLANE_RATE_LIMIT", rateLimit)
			return rate.Limit(rateLimit)
		}
		klog.Warningf("Invalid GOLDILOCKS_CONTROL_PLANE_RATE_LIMIT value '%s', using default 10 calls/second", rateLimitStr)
	}
	klog.Infof("🏃 Control plane rate limit set to default 10 calls/second")
	return 10
}

// GetInstance returns a Reconciler singleton
func GetInstance() *Reconciler {
	if singleton == nil {
		singleton = &Reconciler{
			KubeClient:            kube.GetInstance(),
			VPAClient:             kube.GetVPAInstance(),
			DynamicClient:         kube.GetDynamicInstance(),
			ControllerUtilsClient: kube.GetControllerUtilsInstance(),
			operationSemaphore:    make(chan struct{}, 3), // Allow max 3 concurrent VPA operations
			rateLimiter:           rate.NewLimiter(getControlPlaneRateLimit(), 1), // Configurable rate limit, burst of 1
		}
	}
	return singleton
}

// SetInstance sets the singleton using preconstructed k8s and vpa clients. Used for testing.
func SetInstance(k8s *kube.ClientInstance, vpa *kube.VPAClientInstance, dynamic *kube.DynamicClientInstance, controller *kube.ControllerUtilsClientInstance) *Reconciler {
	singleton = &Reconciler{
		KubeClient:            k8s,
		VPAClient:             vpa,
		DynamicClient:         dynamic,
		ControllerUtilsClient: controller,
		operationSemaphore:    make(chan struct{}, 3), // Allow max 3 concurrent VPA operations
		rateLimiter:           rate.NewLimiter(getControlPlaneRateLimit(), 1), // Configurable rate limit, burst of 1
	}
	return singleton
}

// ReconcileNamespace makes a vpa for every pod controller type in the namespace.
func (r *Reconciler) ReconcileNamespace(namespace *corev1.Namespace) error {
	// Acquire a write lock to prevent new events from being processed during reconciliation
	r.reconciliationMutex.Lock()
	defer r.reconciliationMutex.Unlock()
	
	// Wait for any ongoing VPA operations to complete before starting new reconciliation
	r.activeOperationsWG.Wait()
	
	// Acquire the mutex to prevent concurrent bulk operations that could overwhelm etcd
	r.operationMutex.Lock()
	defer r.operationMutex.Unlock()

	controllerLog.SetLogger(controllerUtilsLogr)
	nsName := namespace.Name
	
	
	klog.V(2).Infof("Starting reconciliation for namespace/%s", nsName)
	
	vpas, err := r.listVPAs(nsName)
	if err != nil {
		klog.Error(err.Error())
		return err
	}

	if !r.namespaceIsManaged(namespace) {
		klog.V(2).Infof("Namespace/%s is not managed, cleaning up VPAs if they exist...", namespace.Name)
		// Namespaced used to be managed, but isn't anymore. Delete all of the
		// VPAs that we control.
		return r.cleanUpManagedVPAsInNamespace(nsName, vpas)
	}

	controllers, err := r.listControllers(nsName)
	if err != nil {
		// Check if this is an RBAC error - just log and continue with empty list
		if utils.IsRBACError(err) {
			klog.V(2).Infof("RBAC permissions prevent controller discovery in namespace %s: %v", nsName, err)
			// Continue with empty controller list - we can still clean up existing VPAs
			controllers = []Controller{}
		} else {
			klog.Error(err.Error())
			return err
		}
	}
	
	// Add debug logging to understand what's happening with cloudflare-operator-system
	if nsName == "cloudflare-operator-system" {
		klog.V(3).Infof("🔍 DEBUG: cloudflare-operator-system processed - found %d controllers, error: %v", len(controllers), err)
	}

	err = r.reconcileControllersAndVPAs(namespace, vpas, controllers)
	if err != nil {
		return err
	}
	
	// Wait for all VPA operations started during this reconciliation to complete
	r.activeOperationsWG.Wait()
	
	klog.V(2).Infof("Completed reconciliation for namespace/%s with all VPA operations finished", nsName)
	return nil
}

func (r *Reconciler) cleanUpManagedVPAsInNamespace(namespace string, vpas []vpav1.VerticalPodAutoscaler) error {
	if len(vpas) < 1 {
		klog.V(4).Infof("No goldilocks managed VPAs found in Namespace/%s, skipping cleanup", namespace)
		return nil
	}
	klog.Infof("Deleting all goldilocks managed VPAs in Namespace/%s", namespace)
	for _, vpa := range vpas {
		// Track this operation in the wait group
		r.activeOperationsWG.Add(1)
		
		// Acquire semaphore to limit concurrent VPA operations
		r.operationSemaphore <- struct{}{}
		err := r.deleteVPA(vpa)
		<-r.operationSemaphore // Release semaphore
		
		// Mark this operation as complete
		r.activeOperationsWG.Done()
		
		if err != nil {
			return err
		}
		
	}
	return nil
}

func (r *Reconciler) namespaceIsManaged(namespace *corev1.Namespace) bool {
	for k, v := range namespace.Labels {
		klog.V(4).Infof("Namespace/%s found label: %s=%s", namespace.Name, k, v)
		if strings.ToLower(k) != utils.VpaEnabledLabel {
			klog.V(9).Infof("Namespace/%s with label key %s does not match enabled label %s", namespace.Name, k, utils.VpaEnabledLabel)
			continue
		}
		enabled, err := strconv.ParseBool(v)
		if err != nil {
			klog.Errorf("Found unsupported value for Namespace/%s label %s=%s, defaulting to false", namespace.Name, k, v)
			return false
		}
		return enabled
	}

	for _, included := range r.IncludeNamespaces {
		if namespace.Name == included {
			return true
		}
	}
	for _, excluded := range r.ExcludeNamespaces {
		if namespace.Name == excluded {
			return false
		}
	}

	return r.OnByDefault
}

func (r *Reconciler) reconcileControllersAndVPAs(ns *corev1.Namespace, vpas []vpav1.VerticalPodAutoscaler, controllers []Controller) error {
	defaultUpdateMode, _ := vpaUpdateModeForResource(ns)
	defaultResourcePolicy, _ := vpaResourcePolicyForResource(ns)
	defaultMinReplicas, _ := vpaMinReplicasForResource(ns)

	// these keys will eventually contain the leftover vpas that do not have a matching controller associated
	vpaHasAssociatedController := map[string]bool{}
	for _, controller := range controllers {
		// Check if the controller kind is in the ignore list
		if lo.Contains(r.IgnoreControllerKind, controller.Kind) {
			continue
		}

		var cvpa *vpav1.VerticalPodAutoscaler
		// search for the matching vpa (will have the same name)
		for idx, vpa := range vpas {
			if fmt.Sprintf("goldilocks-%s", controller.Name) == vpa.Name {
				// found the vpa associated with this controller
				cvpa = &vpas[idx]
				vpaHasAssociatedController[cvpa.Name] = true
				break
			}
		}

		// for logging
		vpaName := "none"
		if cvpa != nil {
			vpaName = cvpa.Name
		}
		klog.V(2).Infof("Reconciling Namespace/%s for %s/%s with VPA/%s", ns.Name, controller.Kind, controller.Name, vpaName)
		
		// Track this operation in the wait group
		r.activeOperationsWG.Add(1)
		
		// Acquire semaphore to limit concurrent VPA operations
		r.operationSemaphore <- struct{}{}
		err := r.reconcileControllerAndVPA(ns, controller, cvpa, defaultUpdateMode, defaultResourcePolicy, defaultMinReplicas)
		<-r.operationSemaphore // Release semaphore
		
		// Mark this operation as complete
		r.activeOperationsWG.Done()
		
		if err != nil {
			return err
		}
		
	}

	for _, vpa := range vpas {
		if !vpaHasAssociatedController[vpa.Name] {
			// these vpas do not have a matching controller, delete them
			klog.V(2).Infof("Deleting dangling VPA/%s in Namespace/%s", vpa.Name, ns.Name)
			
			// Track this operation in the wait group
			r.activeOperationsWG.Add(1)
			
			// Acquire semaphore to limit concurrent VPA operations
			r.operationSemaphore <- struct{}{}
			err := r.deleteVPA(vpa)
			<-r.operationSemaphore // Release semaphore
			
			// Mark this operation as complete
			r.activeOperationsWG.Done()
			
			if err != nil {
				return err
			}
			
		}
	}

	return nil
}

func (r *Reconciler) reconcileControllerAndVPA(ns *corev1.Namespace, controller Controller, vpa *vpav1.VerticalPodAutoscaler, vpaUpdateMode *vpav1.UpdateMode, vpaResourcePolicy *vpav1.PodResourcePolicy, minReplicas *int32) error {
	controllerObj := controller.Unstructured.DeepCopyObject()
	if vpaUpdateModeOverride, explicit := vpaUpdateModeForResource(controllerObj); explicit {
		vpaUpdateMode = vpaUpdateModeOverride
		klog.V(5).Infof("%s/%s has custom vpa-update-mode=%s", controller.Kind, controller.Name, *vpaUpdateMode)
	}

	if vpaResourcePolicyOverride, explicit := vpaResourcePolicyForResource(controllerObj); explicit {
		vpaResourcePolicy = vpaResourcePolicyOverride
		klog.V(5).Infof("%s/%s has custom vpa-resource-policy", controller.Kind, controller.Name)
	}

	desiredVPA := r.getVPAObject(vpa, ns, controller, vpaUpdateMode, vpaResourcePolicy, minReplicas)

	if vpa == nil {
		klog.V(5).Infof("%s/%s does not have a VPA currently, creating VPA/%s", controller.Kind, controller.Name, controller.Name)
		// no vpa exists, create one (use the same name as the controller)
		err := r.createVPA(desiredVPA)
		if err != nil {
			return err
		}
	} else {
		// vpa exists
		klog.V(5).Infof("%s/%s has a VPA currently, updating VPA/%s", controller.Kind, controller.Name, controller.Name)
		err := r.updateVPA(desiredVPA)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) listControllers(namespace string) ([]Controller, error) {
	controllers := []Controller{}
	
	// Rate limit this API call
	ctx, cancel := utils.CreateContextWithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := r.waitForRateLimit(ctx, fmt.Sprintf("list controllers in %s", namespace)); err != nil {
		return nil, err
	}
	
	allTopControllers, err := r.ControllerUtilsClient.Client.GetAllTopControllersSummary(namespace)
	if err != nil {
		return nil, err
	}
	for _, controller := range allTopControllers {
		c := controller.TopController
		if c.GetKind() == "" || c.GetKind() == "Pod" || c.GetAPIVersion() == "" {
			klog.V(5).Infof("No toplevel controller found for pod %s/%s", namespace, c.GetName())
			klog.V(5).Infof("Kind: '%s', APIVersion: '%s'", c.GetKind(), c.GetAPIVersion())
			continue
		}
		controllers = append(controllers, Controller{
			APIVersion:   c.GetAPIVersion(),
			Kind:         c.GetKind(),
			Name:         c.GetName(),
			Unstructured: &c,
		})
	}

	return controllers, nil
}

func (r *Reconciler) listVPAs(namespace string) ([]vpav1.VerticalPodAutoscaler, error) {
	vpaListOptions := metav1.ListOptions{
		LabelSelector: labels.Set(utils.VPALabels).String(),
	}

	var existingVPAs *vpav1.VerticalPodAutoscalerList
	ctx, cancel := utils.CreateContextWithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Rate limit this API call
	if err := r.waitForRateLimit(ctx, fmt.Sprintf("list VPAs in %s", namespace)); err != nil {
		return nil, err
	}
	
	err := utils.RetryWithExponentialBackoff(ctx, func(ctx context.Context) error {
		var listErr error
		existingVPAs, listErr = r.VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(namespace).List(ctx, vpaListOptions)
		return listErr
	}, fmt.Sprintf("list VPAs in namespace %s", namespace))

	if err != nil {
		return nil, err
	}

	klog.V(2).Infof("There are %d vpas in Namespace/%s", len(existingVPAs.Items), namespace)
	if klog.V(9).Enabled() {
		for _, vpa := range existingVPAs.Items {
			klog.V(9).Infof("Found VPA/%s in Namespace/%s", vpa.Name, namespace)
		}
	}

	return existingVPAs.Items, nil
}

func (r *Reconciler) deleteVPA(vpa vpav1.VerticalPodAutoscaler) error {
	if r.DryRun {
		klog.Infof("Not deleting VPA/%s due to dryrun.", vpa.Name)
		return nil
	}

	ctx, cancel := utils.CreateContextWithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Rate limit this API call
	if err := r.waitForRateLimit(ctx, fmt.Sprintf("delete VPA %s/%s", vpa.Namespace, vpa.Name)); err != nil {
		return err
	}

	err := utils.RetryWithExponentialBackoff(ctx, func(ctx context.Context) error {
		return r.VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(vpa.Namespace).Delete(ctx, vpa.Name, metav1.DeleteOptions{})
	}, fmt.Sprintf("delete VPA %s/%s", vpa.Namespace, vpa.Name))

	if err != nil {
		klog.Errorf("Error deleting VPA/%s in Namespace/%s: %v", vpa.Name, vpa.Namespace, err)
		return err
	}
	klog.Infof("Deleted VPA/%s in Namespace/%s", vpa.Name, vpa.Namespace)
	return nil
}

func (r *Reconciler) createVPA(vpa vpav1.VerticalPodAutoscaler) error {
	if !r.DryRun {
		klog.V(9).Infof("Creating VPA/%s: %v", vpa.Name, vpa)

		ctx, cancel := utils.CreateContextWithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		// Rate limit this API call
		if err := r.waitForRateLimit(ctx, fmt.Sprintf("create VPA %s/%s", vpa.Namespace, vpa.Name)); err != nil {
			return err
		}

		err := utils.RetryWithExponentialBackoff(ctx, func(ctx context.Context) error {
			_, createErr := r.VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(vpa.Namespace).Create(ctx, &vpa, metav1.CreateOptions{})
			return createErr
		}, fmt.Sprintf("create VPA %s/%s", vpa.Namespace, vpa.Name))

		if err != nil {
			klog.Errorf("Error creating VPA/%s in Namespace/%s: %v", vpa.Name, vpa.Namespace, err)
			return err
		}
		klog.Infof("Created VPA/%s in Namespace/%s", vpa.Name, vpa.Namespace)
	} else {
		klog.Infof("Not creating VPA/%s in Namespace/%s due to dryrun.", vpa.Name, vpa.Namespace)
	}
	return nil
}

func (r *Reconciler) updateVPA(vpa vpav1.VerticalPodAutoscaler) error {
	if !r.DryRun {
		klog.V(9).Infof("Updating VPA/%s: %v", vpa.Name, vpa)

		ctx, cancel := utils.CreateContextWithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		// Rate limit this API call
		if err := r.waitForRateLimit(ctx, fmt.Sprintf("update VPA %s/%s", vpa.Namespace, vpa.Name)); err != nil {
			return err
		}

		// Use enhanced retry with exponential backoff for etcd failures
		err := utils.RetryWithExponentialBackoff(ctx, func(ctx context.Context) error {
			// For conflict errors, use the original retry logic
			retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				// Note: Normally we're supposed to be getting the current VPA object, then updating that object between
				//       each retry attempt, but since goldilocks should be the only controller that is manipulating
				//       these VPA objects then it's safe to use the desired VPA that is originally passed to this function.
				_, updateErr := r.VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(vpa.Namespace).Update(ctx, &vpa, metav1.UpdateOptions{})
				return updateErr
			})
			return retryErr
		}, fmt.Sprintf("update VPA %s/%s", vpa.Namespace, vpa.Name))

		if err != nil {
			klog.Errorf("Error updating VPA/%s in Namespace/%s: %v", vpa.Name, vpa.Namespace, err)
			return err
		}
		klog.V(2).Infof("Updated VPA/%s in Namespace/%s", vpa.Name, vpa.Namespace)
	} else {
		klog.Infof("Not updating VPA/%s in Namespace/%s due to dryrun.", vpa.Name, vpa.Namespace)
	}
	return nil
}

func (r *Reconciler) getVPAObject(existingVPA *vpav1.VerticalPodAutoscaler, ns *corev1.Namespace, controller Controller, updateMode *vpav1.UpdateMode, resourcePolicy *vpav1.PodResourcePolicy, minReplicas *int32) vpav1.VerticalPodAutoscaler {
	var desiredVPA vpav1.VerticalPodAutoscaler

	// create a brand new vpa with the correct information
	if existingVPA == nil {
		desiredVPA = vpav1.VerticalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "goldilocks-" + controller.Name,
				Namespace: ns.Name,
			},
		}
	} else {
		// or use the existing VPA as a template to update from
		desiredVPA = *existingVPA
	}

	// update the labels on the VPA
	desiredVPA.Labels = utils.VPALabels

	// update the spec on the VPA
	desiredVPA.Spec = vpav1.VerticalPodAutoscalerSpec{
		TargetRef: &autoscaling.CrossVersionObjectReference{
			APIVersion: controller.APIVersion,
			Kind:       controller.Kind,
			Name:       controller.Name,
		},
		UpdatePolicy: &vpav1.PodUpdatePolicy{
			UpdateMode: updateMode,
		},
		ResourcePolicy: resourcePolicy,
	}

	if minReplicas != nil {
		if *minReplicas > 0 {
			desiredVPA.Spec.UpdatePolicy.MinReplicas = minReplicas
		}

	}

	return desiredVPA
}

// vpaUpdateModeForResource searches the resource's annotations and labels for a vpa-update-mode
// key/value and uses that key/value to return the proper UpdateMode type
func vpaUpdateModeForResource(obj runtime.Object) (*vpav1.UpdateMode, bool) {
	requestedVPAMode := vpav1.UpdateModeOff
	explicit := false

	requestStr := ""
	accessor, _ := meta.Accessor(obj)
	if val, ok := accessor.GetAnnotations()[utils.VpaUpdateModeKey]; ok {
		requestStr = val
	} else if val, ok := accessor.GetLabels()[utils.VpaUpdateModeKey]; ok {
		requestStr = val
	}
	if requestStr != "" {
		requestStr = strings.ToUpper(requestStr[0:1]) + strings.ToLower(requestStr[1:])
		requestedVPAMode = vpav1.UpdateMode(requestStr)
		explicit = true
	}

	return &requestedVPAMode, explicit
}

// vpaResourcePolicyForResource get the resource's annotation for the vpa pod resource policy
// key/value and the value is the json definition of the pod resource policy
func vpaResourcePolicyForResource(obj runtime.Object) (*vpav1.PodResourcePolicy, bool) {
	explicit := false

	resourcePolicyStr := ""
	accessor, _ := meta.Accessor(obj)
	if val, ok := accessor.GetAnnotations()[utils.VpaResourcePolicyAnnotation]; ok {
		resourcePolicyStr = val
	} else if val, ok := accessor.GetLabels()[utils.VpaResourcePolicyAnnotation]; ok {
		resourcePolicyStr = val
	}

	if resourcePolicyStr == "" {
		return nil, explicit
	}

	explicit = true
	resourcePol := vpav1.PodResourcePolicy{}
	err := json.NewDecoder(bytes.NewReader([]byte(resourcePolicyStr))).Decode(&resourcePol)
	if err != nil {
		klog.Error(err.Error())
		return nil, explicit
	}

	return &resourcePol, explicit
}

// vpaMinReplicas sets the VPA minimum replicas required for eviction
func vpaMinReplicasForResource(obj runtime.Object) (*int32, bool) {
	explicit := false

	minReplicasString := ""
	accessor, _ := meta.Accessor(obj)
	if val, ok := accessor.GetAnnotations()[utils.VpaMinReplicasAnnotation]; ok {
		minReplicasString = val
	} else if val, ok := accessor.GetLabels()[utils.VpaMinReplicasAnnotation]; ok {
		minReplicasString = val
	}

	if minReplicasString == "" {
		return nil, explicit
	}

	explicit = true
	minReplicas, err := strconv.ParseInt(minReplicasString, 10, 32)
	if err != nil {
		klog.Error(err.Error())
		return nil, explicit
	}

	minReplicasInt32 := int32(minReplicas)

	return &minReplicasInt32, explicit
}

// IsReconciliationInProgress returns true if VPA reconciliation is currently active
func (r *Reconciler) IsReconciliationInProgress() bool {
	// Try to acquire a read lock - if we can't, reconciliation is in progress
	acquired := r.reconciliationMutex.TryRLock()
	if acquired {
		r.reconciliationMutex.RUnlock()
		return false
	}
	return true
}

// WaitForReconciliationToComplete blocks until any active reconciliation completes
func (r *Reconciler) WaitForReconciliationToComplete() {
	// Acquire and immediately release a read lock to wait for active reconciliation
	r.reconciliationMutex.RLock()
	defer r.reconciliationMutex.RUnlock()
}



// waitForRateLimit enforces the 10 calls/second rate limit for control plane operations
func (r *Reconciler) waitForRateLimit(ctx context.Context, operation string) error {
	if err := r.rateLimiter.Wait(ctx); err != nil {
		klog.V(3).Infof("⏳ RATE LIMIT: %s operation delayed due to rate limiting: %v", operation, err)
		return err
	}
	return nil
}
