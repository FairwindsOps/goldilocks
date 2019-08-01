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

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	v1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2"
	"k8s.io/klog"

	"github.com/fairwindsops/goldilocks/pkg/kube"
	"github.com/fairwindsops/goldilocks/pkg/utils"
)

type containerSummary struct {
	LowerBound    v1.ResourceList `json:"lowerBound"`
	UpperBound    v1.ResourceList `json:"upperBound"`
	Limits        v1.ResourceList `json:"limits"`
	Requests      v1.ResourceList `json:"requests"`
	ContainerName string          `json:"containerName"`
}

type deploymentSummary struct {
	Containers     []containerSummary `json:"containers"`
	DeploymentName string             `json:"deploymentName"`
	Namespace      string             `json:"namespace"`
}

// Summary struct is for storing a summary of recommendation data.
type Summary struct {
	Deployments []deploymentSummary `json:"deployments"`
	Namespaces  []string            `json:"namespaces"`
}

// Run creates a summary of the vpa info for all namespaces.
func Run(vpaLabels map[string]string, excludeContainers string) (Summary, error) {
	klog.V(3).Infof("Looking for VPAs with labels: %v", vpaLabels)

	kubeClientVPA := kube.GetVPAInstance()

	vpaListOptions := metav1.ListOptions{
		LabelSelector: labels.Set(vpaLabels).String(),
	}

	vpas, err := kubeClientVPA.Client.AutoscalingV1beta2().VerticalPodAutoscalers("").List(vpaListOptions)
	if err != nil {
		klog.Error(err.Error())
	}
	klog.V(10).Infof("Found vpas: %v", vpas)

	summary, _ := constructSummary(vpas, excludeContainers)
	return summary, nil
}

func constructSummary(vpas *v1beta2.VerticalPodAutoscalerList, excludeContainers string) (Summary, error) {
	var summary Summary
	if len(vpas.Items) <= 0 {
		return summary, nil
	}

	kubeClient := kube.GetInstance()

	containerExclusions := strings.Split(excludeContainers, ",")

	for _, vpa := range vpas.Items {
		klog.V(8).Infof("Analyzing vpa: %v", vpa.ObjectMeta.Name)

		var deploy deploymentSummary
		deploy.DeploymentName = vpa.ObjectMeta.Name
		deploy.Namespace = vpa.ObjectMeta.Namespace

		summary.Namespaces = append(summary.Namespaces, deploy.Namespace)

		deployment, err := kubeClient.Client.AppsV1().Deployments(deploy.Namespace).Get(deploy.DeploymentName, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("Error retrieving deployment from API: %v", err)
		}

		if vpa.Status.Recommendation == nil {
			klog.V(2).Infof("Empty status on %v", deploy.DeploymentName)
			break
		}
		if len(vpa.Status.Recommendation.ContainerRecommendations) <= 0 {
			klog.V(2).Infof("No recommendations found in the %v vpa.", deploy.DeploymentName)
			break
		}

		if annotationValue, annotationFound := deployment.Annotations["goldilocks.fairwinds.com/exclude-containers"]; annotationFound {
			containerExclusions = append(containerExclusions, strings.Split(annotationValue, ",")...)
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
					ContainerName: containerRecommendation.ContainerName,
					UpperBound:    containerRecommendation.UpperBound,
					LowerBound:    containerRecommendation.LowerBound,
				}
				if c.Name == containerRecommendation.ContainerName {
					klog.V(6).Infof("Resources for %s: %v", c.Name, c.Resources)
					container.Limits = c.Resources.Limits
					container.Requests = c.Resources.Requests
				}
			}

			deploy.Containers = append(deploy.Containers, container)
		}
		summary.Deployments = append(summary.Deployments, deploy)
	}

	summary.Namespaces = utils.UniqueString(summary.Namespaces)
	return summary, nil
}
