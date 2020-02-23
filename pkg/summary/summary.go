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

package summary

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	v1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2"
	"k8s.io/klog"

	"github.com/fairwindsops/goldilocks/pkg/kube"
	"github.com/fairwindsops/goldilocks/pkg/utils"
)

type containerSummary struct {
	LowerBound     corev1.ResourceList `json:"lowerBound"`
	UpperBound     corev1.ResourceList `json:"upperBound"`
	Target         corev1.ResourceList `json:"target"`
	UncappedTarget corev1.ResourceList `json:"uncappedTarget"`
	Limits         corev1.ResourceList `json:"limits"`
	Requests       corev1.ResourceList `json:"requests"`
	ContainerName  string              `json:"containerName"`
}

type resourceSummary struct {
	Containers   []containerSummary `json:"containers"`
	ResourceName string             `json:"resourceName"`
	Kind         string             `json:"kind"`
	Namespace    string             `json:"namespace"`
}

// Summary struct is for storing a summary of recommendation data.
type Summary struct {
	Resources  []resourceSummary `json:"resources"`
	Namespaces []string          `json:"namespaces"`
}

// Client checks if VPA objects should be created or deleted
// what? how does it do that? it checks it??
type Client struct {
	//changing these two to match the naming in here...but should it be consistent?
	KubeClient    *kube.ClientInstance
	KubeClientVPA *kube.VPAClientInstance
}

var singleton *Client

// GetInstance returns a Client singleton
func GetInstance() *Client {
	if singleton == nil {
		singleton = &Client{
			KubeClient:    kube.GetInstance(),
			KubeClientVPA: kube.GetVPAInstance(),
		}
	}
	return singleton
}

// SetInstance sets the singleton using preconstructed k8s and vpa clients. Used for testing.
func SetInstance(k8s *kube.ClientInstance, vpa *kube.VPAClientInstance) *Client {
	singleton = &Client{
		KubeClient:    k8s,
		KubeClientVPA: vpa,
	}
	return singleton
}

// Run creates a summary of the vpa info for all namespaces.
func (client *Client) Run(vpaLabels map[string]string, excludeContainers string) (Summary, error) {
	klog.V(3).Infof("Looking for VPAs with labels: %v", vpaLabels)

	vpaListOptions := metav1.ListOptions{
		LabelSelector: labels.Set(vpaLabels).String(),
	}

	vpas, err := client.KubeClientVPA.Client.AutoscalingV1beta2().VerticalPodAutoscalers("").List(vpaListOptions)
	if err != nil {
		klog.Error(err.Error())
	}
	klog.V(10).Infof("Found vpas: %v", vpas)

	summary, _ := GetInstance().constructSummary(vpas, excludeContainers)
	return summary, nil
}

func (client *Client) constructSummary(vpas *v1beta2.VerticalPodAutoscalerList, excludeContainers string) (Summary, error) {
	var summary Summary
	if len(vpas.Items) <= 0 {
		return summary, nil
	}

	containerExclusions := strings.Split(excludeContainers, ",")

	for _, vpa := range vpas.Items {
		klog.V(8).Infof("Analyzing %v vpa: %v", vpa.Spec.TargetRef.Kind, vpa.ObjectMeta.Name)

		var resource resourceSummary
		resource.ResourceName = vpa.ObjectMeta.Name
		resource.Namespace = vpa.ObjectMeta.Namespace
		resource.Kind = vpa.Spec.TargetRef.Kind

		summary.Namespaces = append(summary.Namespaces, resource.Namespace)

		if vpa.Status.Recommendation == nil {
			klog.V(2).Infof("Empty status on %v", resource.ResourceName)
			continue
		}
		if len(vpa.Status.Recommendation.ContainerRecommendations) <= 0 {
			klog.V(2).Infof("No recommendations found in the %v vpa.", resource.ResourceName)
			continue
		}

		switch resource.Kind {
		case "Deployment":
			deploymentSummary(client, vpa, &resource, containerExclusions)
		case "DaemonSet":
			daemonsetSummary(client, vpa, &resource, containerExclusions)
		}
		summary.Resources = append(summary.Resources, resource)
	}

	summary.Namespaces = utils.UniqueString(summary.Namespaces)
	return summary, nil
}

func deploymentSummary(client *Client, vpa v1beta2.VerticalPodAutoscaler, resource *resourceSummary, containerExclusions []string) {
	deployment, err := client.KubeClient.Client.AppsV1().Deployments(resource.Namespace).Get(resource.ResourceName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Error retrieving deployment from API: %v", err)
	}

	if labelValue, labelFound := deployment.Labels["goldilocks.fairwinds.com/exclude-containers"]; labelFound {
		containerExclusions = append(containerExclusions, strings.Split(labelValue, ",")...)
	}

CONTAINER_REC_LOOP:
	for _, containerRecommendation := range vpa.Status.Recommendation.ContainerRecommendations {
		for _, exclusion := range containerExclusions {
			if exclusion == containerRecommendation.ContainerName {
				klog.V(2).Infof("Excluding container %v", containerRecommendation.ContainerName)
				continue CONTAINER_REC_LOOP
			}
		}

		var container containerSummary
		for _, c := range deployment.Spec.Template.Spec.Containers {
			container = containerSummary{
				ContainerName:  containerRecommendation.ContainerName,
				UpperBound:     containerRecommendation.UpperBound,
				LowerBound:     containerRecommendation.LowerBound,
				Target:         containerRecommendation.Target,
				UncappedTarget: containerRecommendation.UncappedTarget,
			}
			if c.Name == containerRecommendation.ContainerName {
				klog.V(6).Infof("Resources for %s: %v", c.Name, c.Resources)
				container.Limits = c.Resources.Limits
				container.Requests = c.Resources.Requests
			}
		}

		resource.Containers = append(resource.Containers, container)
	}
}

func daemonsetSummary(client *Client, vpa v1beta2.VerticalPodAutoscaler, resource *resourceSummary, containerExclusions []string) {
	daemonset, err := client.KubeClient.Client.AppsV1().DaemonSets(resource.Namespace).Get(resource.ResourceName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Error retrieving daemonset from API: %v", err)
	}

	if labelValue, labelFound := daemonset.Labels["goldilocks.fairwinds.com/exclude-containers"]; labelFound {
		containerExclusions = append(containerExclusions, strings.Split(labelValue, ",")...)
	}

CONTAINER_REC_LOOP:
	for _, containerRecommendation := range vpa.Status.Recommendation.ContainerRecommendations {
		for _, exclusion := range containerExclusions {
			if exclusion == containerRecommendation.ContainerName {
				klog.V(2).Infof("Excluding container %v", containerRecommendation.ContainerName)
				continue CONTAINER_REC_LOOP
			}
		}

		var container containerSummary
		for _, c := range daemonset.Spec.Template.Spec.Containers {
			container = containerSummary{
				ContainerName:  containerRecommendation.ContainerName,
				UpperBound:     containerRecommendation.UpperBound,
				LowerBound:     containerRecommendation.LowerBound,
				Target:         containerRecommendation.Target,
				UncappedTarget: containerRecommendation.UncappedTarget,
			}
			if c.Name == containerRecommendation.ContainerName {
				klog.V(6).Infof("Resources for %s: %v", c.Name, c.Resources)
				container.Limits = c.Resources.Limits
				container.Requests = c.Resources.Requests
			}
		}

		resource.Containers = append(resource.Containers, container)
	}
}
