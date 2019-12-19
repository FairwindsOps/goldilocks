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
	"strconv"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	autoscaling "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	v1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2"
	"k8s.io/klog"

	"github.com/fairwindsops/goldilocks/pkg/kube"
	"github.com/fairwindsops/goldilocks/pkg/utils"
)

// Reconciler checks if VPA objects should be created or deleted
type Reconciler struct {
	KubeClient        *kube.ClientInstance
	VPAClient         *kube.VPAClientInstance
	OnByDefault       bool
	DryRun            bool
	IncludeNamespaces []string
	ExcludeNamespaces []string
}

var singleton *Reconciler

// GetInstance returns a Reconciler singleton
func GetInstance() *Reconciler {
	if singleton == nil {
		singleton = &Reconciler{
			KubeClient: kube.GetInstance(),
			VPAClient:  kube.GetVPAInstance(),
		}
	}
	return singleton
}

// SetInstance sets the singleton using preconstructed k8s and vpa clients. Used for testing.
func SetInstance(k8s *kube.ClientInstance, vpa *kube.VPAClientInstance) *Reconciler {
	singleton = &Reconciler{
		KubeClient: k8s,
		VPAClient:  vpa,
	}
	return singleton
}

// ReconcileNamespace makes a vpa for every deployment in the namespace.
// Check if deployment has label for false before applying vpa.
func (r Reconciler) ReconcileNamespace(namespace *corev1.Namespace) error {
	nsName := namespace.ObjectMeta.Name
	vpas, err := r.listVPAs(nsName)
	if err != nil {
		klog.Error(err.Error())
		return err
	}

	if !r.namespaceIsManaged(namespace) {
		klog.V(2).Infof("Namespace/%s is not managed, cleaning up VPAs...", namespace.Name)
		// Namespaced used to be managed, but isn't anymore. Delete all of the
		// VPAs that we control.
		return r.cleanUpManagedVPAsInNamespace(nsName, vpas)
	}

	deployments, err := r.listDeployments(nsName)
	if err != nil {
		klog.Error(err.Error())
		return err
	}

	return r.reconcileDeploymentsAndVPAs(namespace, vpas, deployments)
}

func (r Reconciler) cleanUpManagedVPAsInNamespace(namespace string, vpas []v1beta2.VerticalPodAutoscaler) error {
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

// NOTE: This is not used right now.  Deployments have been scrapped.
// Keeping this here for future development.
func (r Reconciler) checkDeploymentLabels(deployment *appsv1.Deployment) (bool, error) {
	if len(deployment.ObjectMeta.Labels) > 0 {
		for k, v := range deployment.ObjectMeta.Labels {
			klog.V(7).Infof("Deployment Label - %s: %s", k, v)
			if strings.ToLower(k) == utils.VpaEnabledLabel {
				return strconv.ParseBool(v)
			}
		}
	}
	return false, nil
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

func (r Reconciler) reconcileDeploymentsAndVPAs(ns *corev1.Namespace, vpas []v1beta2.VerticalPodAutoscaler, deployments []appsv1.Deployment) error {
	// these keys will eventually contain the leftover vpas that do not have a matching deployment associated
	vpaHasAssociatedDeployment := map[string]bool{}
	for _, deployment := range deployments {
		var dvpa *v1beta2.VerticalPodAutoscaler
		// search for the matching vpa (will have the same name)
		for idx, vpa := range vpas {
			if deployment.Name == vpa.Name {
				// found the vpa associated with this deployment
				dvpa = &vpas[idx]
				vpaHasAssociatedDeployment[dvpa.Name] = true
				break
			}
		}

		// for logging
		vpaName := "none"
		if dvpa != nil {
			vpaName = dvpa.Name
		}
		klog.V(2).Infof("Reconciling Namespace/%s for Deployment/%s with VPA/%s", ns.Name, deployment.Name, vpaName)
		err := r.reconcileDeploymentAndVPA(ns, deployment, dvpa)
		if err != nil {
			return err
		}
	}

	for _, vpa := range vpas {
		if !vpaHasAssociatedDeployment[vpa.Name] {
			// these vpas do not have a matching deployment, delete them
			klog.V(2).Infof("Deleting dangling VPA/%s in Namespace/%s", vpa.Name, ns.Name)
			err := r.deleteVPA(vpa)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (r Reconciler) reconcileDeploymentAndVPA(ns *corev1.Namespace, deployment appsv1.Deployment, vpa *v1beta2.VerticalPodAutoscaler) error {
	// get the desiredVPA as configured by annotations on the Namespace
	desiredVPA := r.getVPAObject(ns, deployment.Name)

	// check if the Deployment has its own vpa-update-mode set
	if _, ok := deployment.GetAnnotations()[utils.VpaUpdateModeKey]; ok {
		vpaUpdateMode := vpaUpdateModeForResource(&deployment)
		desiredVPA.Spec.UpdatePolicy.UpdateMode = vpaUpdateMode
		klog.V(5).Infof("Deployment/%s has custom vpa-update-mode=%s", deployment.Name, vpaUpdateMode)
	}

	if vpa == nil {
		klog.V(5).Infof("Deployment/%s does not have a VPA currently, creating VPA/%s", deployment.Name, deployment.Name)
		// no vpa exists, create one (use the same name as the deployment)
		err := r.createVPA(desiredVPA)
		if err != nil {
			return err
		}
	} else {
		// vpa exists
		klog.V(5).Infof("Deployment/%s has a VPA currently, updating VPA/%s", deployment.Name, deployment.Name)
		err := r.updateVPA(desiredVPA)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r Reconciler) listDeployments(namespace string) ([]appsv1.Deployment, error) {
	deployments, err := r.KubeClient.Client.AppsV1().Deployments(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	klog.V(2).Infof("There are %d deployments in Namespace/%s", len(deployments.Items), namespace)
	if klog.V(9) {
		for _, d := range deployments.Items {
			klog.V(9).Infof("Found Deployment/%s in Namespace/%s", d.Name, namespace)
		}
	}

	return deployments.Items, nil
}

func (r Reconciler) listVPAs(namespace string) ([]v1beta2.VerticalPodAutoscaler, error) {
	vpaListOptions := metav1.ListOptions{
		LabelSelector: labels.Set(utils.VPALabels).String(),
	}
	existingVPAs, err := r.VPAClient.Client.AutoscalingV1beta2().VerticalPodAutoscalers(namespace).List(vpaListOptions)
	if err != nil {
		return nil, err
	}

	klog.V(2).Infof("There are %d vpas in Namespace/%s", len(existingVPAs.Items), namespace)
	if klog.V(9) {
		for _, vpa := range existingVPAs.Items {
			klog.V(9).Infof("Found VPA/%s in Namespace/%s", vpa.Name, namespace)
		}
	}

	return existingVPAs.Items, nil
}

func (r Reconciler) deleteVPA(vpa v1beta2.VerticalPodAutoscaler) error {
	if r.DryRun {
		klog.Infof("Not deleting VPA/%s due to dryrun.", vpa.Name)
		return nil
	}
	deleteOptions := metav1.NewDeleteOptions(0)
	errDelete := r.VPAClient.Client.AutoscalingV1beta2().VerticalPodAutoscalers(vpa.Namespace).Delete(vpa.Name, deleteOptions)
	if errDelete != nil {
		klog.Errorf("Error deleting VPA/%s in Namespace/%s: %v", vpa.Name, vpa.Namespace, errDelete)
		return errDelete
	}
	klog.Infof("Deleted VPA/%s in Namespace/%s", vpa.Name, vpa.Namespace)
	return nil
}

func (r Reconciler) createVPA(vpa v1beta2.VerticalPodAutoscaler) error {
	if !r.DryRun {
		klog.V(9).Infof("Creating VPA/%s: %v", vpa.Name, vpa)
		_, err := r.VPAClient.Client.AutoscalingV1beta2().VerticalPodAutoscalers(vpa.Namespace).Create(&vpa)
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

func (r Reconciler) updateVPA(vpa v1beta2.VerticalPodAutoscaler) error {
	if !r.DryRun {
		klog.V(9).Infof("Updating VPA/%s: %v", vpa.Name, vpa)
		_, err := r.VPAClient.Client.AutoscalingV1beta2().VerticalPodAutoscalers(vpa.Namespace).Update(&vpa)
		if err != nil {
			klog.Errorf("Error updating VPA/%s in Namespace/%s: %v", vpa.Name, vpa.Namespace, err)
			return err
		}
		klog.Infof("Updated VPA/%s in Namespace/%s", vpa.Name, vpa.Namespace)
	} else {
		klog.Infof("Not updating VPA/%s in Namespace/%s due to dryrun.", vpa.Name, vpa.Namespace)
	}
	return nil
}

func (r Reconciler) getVPAObject(ns *corev1.Namespace, vpaName string) v1beta2.VerticalPodAutoscaler {
	return v1beta2.VerticalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vpaName,
			Labels:    utils.VPALabels,
			Namespace: ns.Name,
		},
		Spec: v1beta2.VerticalPodAutoscalerSpec{
			TargetRef: &autoscaling.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       vpaName,
			},
			UpdatePolicy: &v1beta2.PodUpdatePolicy{
				UpdateMode: vpaUpdateModeForResource(ns),
			},
		},
	}
}

// vpaUpdateModeForResource searches the resource's annotations and labels for a vpa-update-mode
// key/value and uses that key/value to return the proper UpdateMode type
func vpaUpdateModeForResource(obj runtime.Object) *v1beta2.UpdateMode {
	var requestedVPAMode string

	// check for vpa-update-mode in annotations first
	accessor, _ := meta.Accessor(obj)
	if val, ok := accessor.GetAnnotations()[utils.VpaUpdateModeKey]; ok {
		requestedVPAMode = val
	} else {
		// check for vpa-update-mode in labels
		for k, v := range accessor.GetLabels() {
			if strings.ToLower(k) != utils.VpaUpdateModeKey {
				continue
			}

			requestedVPAMode = v
		}
	}

	// See: https://github.com/kubernetes/autoscaler/blob/master/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2/types.go#L101
	var updateMode v1beta2.UpdateMode
	switch strings.ToLower(requestedVPAMode) {
	case "off":
		updateMode = v1beta2.UpdateModeOff
	case "auto":
		updateMode = v1beta2.UpdateModeAuto
	case "initial":
		updateMode = v1beta2.UpdateModeInitial
	case "recreate":
		updateMode = v1beta2.UpdateModeRecreate
	default:
		klog.Warningf("Found unsupported value for vpa-update-mode: %s, using default vpa-update-mode=off", requestedVPAMode)
		updateMode = v1beta2.UpdateModeOff
	}

	return &updateMode
}
