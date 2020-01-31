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
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	autoscaling "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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

// NOTE: This is not used right now.  Deployments have been scrapped.
// Keeping this here for future development.
func (vpa Reconciler) checkDeploymentLabels(deployment *appsv1.Deployment) (bool, error) {
	if len(deployment.ObjectMeta.Labels) > 0 {
		for k, v := range deployment.ObjectMeta.Labels {
			klog.V(7).Infof("Deployment Label - %s: %s", k, v)
			if strings.ToLower(k) == "goldilocks.fairwinds.com/enabled" {
				if strings.ToLower(v) == "true" {
					return true, nil
				}
				if strings.ToLower(v) == "false" {
					return false, nil
				}
			}
		}
	}
	return false, nil
}

func (vpa Reconciler) checkNamespaceLabel(namespace *corev1.Namespace) bool {
	for k, v := range namespace.ObjectMeta.Labels {
		klog.V(7).Infof("Namespace label - %s: %s", k, v)
		if strings.ToLower(k) != "goldilocks.fairwinds.com/enabled" {
			continue
		}
		v = strings.ToLower(v)
		if v == "true" {
			return true
		} else if v == "false" {
			return false
		}
		klog.V(2).Infof("Unknown label value on namespace %s: %s=%s", namespace.ObjectMeta.Name, k, v)
	}

	for _, included := range vpa.IncludeNamespaces {
		if namespace.ObjectMeta.Name == included {
			return true
		}
	}
	for _, excluded := range vpa.ExcludeNamespaces {
		if namespace.ObjectMeta.Name == excluded {
			return false
		}
	}

	return vpa.OnByDefault
}

// ReconcileNamespace makes a vpa for every deployment in the namespace.
// Check if deployment has label for false before applying vpa.
func (vpa Reconciler) ReconcileNamespace(namespace *corev1.Namespace, dryrun bool) error {
	nsName := namespace.ObjectMeta.Name
	vpaNames := listVPA(vpa.VPAClient, nsName)

	if create := vpa.checkNamespaceLabel(namespace); !create {
		// Get the list of VPAs that already exist
		if len(vpaNames) < 1 {
			klog.V(4).Infof("No labels and no vpas in namespace. Nothing to do.")
			return nil
		}
		klog.Infof("Deleting all owned VPAs in namespace: %s", namespace)
		for _, vpaName := range vpaNames {
			err := deleteVPA(vpa.VPAClient, nsName, vpaName, dryrun)
			if err != nil {
				return err
			}
		}
		return nil
	}

	//Get the list of deployments in the namespace
	deployments, err := vpa.KubeClient.Client.AppsV1().Deployments(nsName).List(metav1.ListOptions{})
	if err != nil {
		klog.Error(err.Error())
		return err
	}
	var deploymentNames []string
	klog.V(2).Infof("There are %d deployments in the namespace", len(deployments.Items))
	for _, deployment := range deployments.Items {
		deploymentNames = append(deploymentNames, deployment.ObjectMeta.Name)
		klog.V(5).Infof("Found Deployment: %v", deployment.ObjectMeta.Name)
	}

	// Create any VPAs that need to be
	vpaNeeded := utils.Difference(deploymentNames, vpaNames)
	klog.V(3).Infof("Diff deployments, vpas: %v", vpaNeeded)

	if len(vpaNeeded) == 0 {
		klog.Info("All VPAs are in sync.")
	} else if len(vpaNeeded) > 0 {
		for _, vpaName := range vpaNeeded {
			err := createVPA(vpa.VPAClient, nsName, vpaName, dryrun)
			if err != nil {
				return err
			}
		}
	}

	// Now that this is one, we can delete any VPAs that don't have matching deployments.
	vpaDelete := utils.Difference(vpaNames, deploymentNames)
	klog.V(5).Infof("Diff vpas, deployments: %v", vpaDelete)

	if len(vpaDelete) == 0 {
		klog.Info("No VPAs to delete.")
	} else if len(vpaDelete) > 0 {
		for _, vpaName := range vpaDelete {
			err := deleteVPA(vpa.VPAClient, nsName, vpaName, dryrun)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func listVPA(vpaClient *kube.VPAClientInstance, namespace string) []string {

	vpaListOptions := metav1.ListOptions{
		LabelSelector: labels.Set(utils.VpaLabels).String(),
	}

	existingVPAs, err := vpaClient.Client.AutoscalingV1beta2().VerticalPodAutoscalers(namespace).List(vpaListOptions)
	if err != nil {
		klog.Error(err.Error())
		return nil
	}
	var vpaNames []string

	for _, vpa := range existingVPAs.Items {
		vpaNames = append(vpaNames, vpa.ObjectMeta.Name)
		klog.V(5).Infof("Found existing vpa: %v", vpa.ObjectMeta.Name)
	}
	return vpaNames
}

func deleteVPA(vpaClient *kube.VPAClientInstance, namespace string, vpaName string, dryrun bool) error {

	if dryrun {
		klog.Infof("Not deleting %s due to dryrun.", vpaName)
		return nil
	}
	deleteOptions := metav1.NewDeleteOptions(0)
	errDelete := vpaClient.Client.AutoscalingV1beta2().VerticalPodAutoscalers(namespace).Delete(vpaName, deleteOptions)
	if errDelete != nil {
		klog.Errorf("Error deleting vpa: %v", errDelete)
		return errDelete
	}
	klog.Infof("Deleted vpa: %s", vpaName)
	return nil
}

func createVPA(vpaClient *kube.VPAClientInstance, namespace string, vpaName string, dryrun bool) error {
	updateMode := v1beta2.UpdateModeOff
	vpa := &v1beta2.VerticalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:   vpaName,
			Labels: utils.VpaLabels,
		},
		Spec: v1beta2.VerticalPodAutoscalerSpec{
			TargetRef: &autoscaling.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       vpaName,
			},
			UpdatePolicy: &v1beta2.PodUpdatePolicy{
				UpdateMode: &updateMode,
			},
		},
	}

	if !dryrun {
		klog.Infof("Creating vpa: %s", vpaName)
		klog.V(9).Infof("%v", vpa)
		_, err := vpaClient.Client.AutoscalingV1beta2().VerticalPodAutoscalers(namespace).Create(vpa)
		if err != nil {
			klog.Errorf("Error creating vpa: %v", err)
			return err
		}
	} else {
		klog.Infof("Dry run was set. Not creating vpa: %v", vpaName)
	}
	return nil
}
