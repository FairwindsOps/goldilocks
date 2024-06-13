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
	"strconv"
	"strings"

	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2/klogr"

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
	ignoreControllerKind  []string
}

type Controller struct {
	APIVersion   string
	Kind         string
	Name         string
	Unstructured *unstructured.Unstructured
}

var singleton *Reconciler
var controllerUtilsLogr = klogr.New()

// GetInstance returns a Reconciler singleton
func GetInstance() *Reconciler {
	if singleton == nil {
		singleton = &Reconciler{
			KubeClient:            kube.GetInstance(),
			VPAClient:             kube.GetVPAInstance(),
			DynamicClient:         kube.GetDynamicInstance(),
			ControllerUtilsClient: kube.GetControllerUtilsInstance(),
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
	}
	return singleton
}

// ReconcileNamespace makes a vpa for every pod controller type in the namespace.
func (r Reconciler) ReconcileNamespace(namespace *corev1.Namespace) error {
	controllerLog.SetLogger(controllerUtilsLogr)
	nsName := namespace.ObjectMeta.Name
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
		klog.Error(err.Error())
		return err
	}

	return r.reconcileControllersAndVPAs(namespace, vpas, controllers, r.ignoreControllerKind)
}

func (r Reconciler) cleanUpManagedVPAsInNamespace(namespace string, vpas []vpav1.VerticalPodAutoscaler) error {
	if len(vpas) < 1 {
		klog.V(4).Infof("No goldilocks managed VPAs found in Namespace/%s, skipping cleanup", namespace)
		return nil
	}
	klog.Infof("Deleting all goldilocks managed VPAs in Namespace/%s", namespace)
	for _, vpa := range vpas {
		err := r.deleteVPA(vpa)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r Reconciler) namespaceIsManaged(namespace *corev1.Namespace) bool {
	for k, v := range namespace.ObjectMeta.Labels {
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
		if namespace.ObjectMeta.Name == included {
			return true
		}
	}
	for _, excluded := range r.ExcludeNamespaces {
		if namespace.ObjectMeta.Name == excluded {
			return false
		}
	}

	return r.OnByDefault
}

func (r Reconciler) reconcileControllersAndVPAs(ns *corev1.Namespace, vpas []vpav1.VerticalPodAutoscaler, controllers []Controller, ignoreControllerKind []string) error {
	defaultUpdateMode, _ := vpaUpdateModeForResource(ns)
	defaultResourcePolicy, _ := vpaResourcePolicyForResource(ns)
	defaultMinReplicas, _ := vpaMinReplicasForResource(ns)

	// these keys will eventually contain the leftover vpas that do not have a matching controller associated
	vpaHasAssociatedController := map[string]bool{}
	for _, controller := range controllers {
		// Check if the controller kind is in the ignore list
		if contains(ignoreControllerKind, controller.Kind) {
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
		err := r.reconcileControllerAndVPA(ns, controller, cvpa, defaultUpdateMode, defaultResourcePolicy, defaultMinReplicas)
		if err != nil {
			return err
		}
	}

	for _, vpa := range vpas {
		if !vpaHasAssociatedController[vpa.Name] {
			// these vpas do not have a matching controller, delete them
			klog.V(2).Infof("Deleting dangling VPA/%s in Namespace/%s", vpa.Name, ns.Name)
			err := r.deleteVPA(vpa)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Helper function to check if a string is in a slice, used for checking if a controller kind is in the ignore list
func contains(slice []string, item string) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}

func (r Reconciler) reconcileControllerAndVPA(ns *corev1.Namespace, controller Controller, vpa *vpav1.VerticalPodAutoscaler, vpaUpdateMode *vpav1.UpdateMode, vpaResourcePolicy *vpav1.PodResourcePolicy, minReplicas *int32) error {
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

func (r Reconciler) listControllers(namespace string) ([]Controller, error) {
	controllers := []Controller{}
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

func (r Reconciler) listVPAs(namespace string) ([]vpav1.VerticalPodAutoscaler, error) {
	vpaListOptions := metav1.ListOptions{
		LabelSelector: labels.Set(utils.VPALabels).String(),
	}
	existingVPAs, err := r.VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(namespace).List(context.TODO(), vpaListOptions)
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

func (r Reconciler) deleteVPA(vpa vpav1.VerticalPodAutoscaler) error {
	if r.DryRun {
		klog.Infof("Not deleting VPA/%s due to dryrun.", vpa.Name)
		return nil
	}

	errDelete := r.VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(vpa.Namespace).Delete(context.TODO(), vpa.Name, metav1.DeleteOptions{})
	if errDelete != nil {
		klog.Errorf("Error deleting VPA/%s in Namespace/%s: %v", vpa.Name, vpa.Namespace, errDelete)
		return errDelete
	}
	klog.Infof("Deleted VPA/%s in Namespace/%s", vpa.Name, vpa.Namespace)
	return nil
}

func (r Reconciler) createVPA(vpa vpav1.VerticalPodAutoscaler) error {
	if !r.DryRun {
		klog.V(9).Infof("Creating VPA/%s: %v", vpa.Name, vpa)
		_, err := r.VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(vpa.Namespace).Create(context.TODO(), &vpa, metav1.CreateOptions{})
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

func (r Reconciler) updateVPA(vpa vpav1.VerticalPodAutoscaler) error {
	if !r.DryRun {
		klog.V(9).Infof("Updating VPA/%s: %v", vpa.Name, vpa)
		// attempt to update the vpa using retries and backoffs
		// [See: https://github.com/kubernetes/client-go/blob/master/examples/create-update-delete-deployment/main.go#L125]
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			// Note: Normally we're supposed to be getting the current VPA object, then updating that object between
			//       each retry attempt, but since goldilocks should be the only controller that is manipulating
			//       these VPA objects then it's safe to use the desired VPA that is originally passed to this function.
			_, err := r.VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(vpa.Namespace).Update(context.TODO(), &vpa, metav1.UpdateOptions{})
			return err
		})
		if retryErr != nil {
			klog.Errorf("Error updating VPA/%s in Namespace/%s: %v", vpa.Name, vpa.Namespace, retryErr)
			return retryErr
		}
		klog.V(2).Infof("Updated VPA/%s in Namespace/%s", vpa.Name, vpa.Namespace)
	} else {
		klog.Infof("Not updating VPA/%s in Namespace/%s due to dryrun.", vpa.Name, vpa.Namespace)
	}
	return nil
}

func (r Reconciler) getVPAObject(existingVPA *vpav1.VerticalPodAutoscaler, ns *corev1.Namespace, controller Controller, updateMode *vpav1.UpdateMode, resourcePolicy *vpav1.PodResourcePolicy, minReplicas *int32) vpav1.VerticalPodAutoscaler {
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
