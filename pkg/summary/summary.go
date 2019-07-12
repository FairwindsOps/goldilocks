// Copyright 2019 Fairwinds
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
	"k8s.io/klog"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/fairwindsops/goldilocks/pkg/kube"
)

type containerSummary struct {
	LowerBound    v1.ResourceList `json:"lowerBound"`
	UpperBound    v1.ResourceList `json:"upperBound"`
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
}

// Run creates a summary of the vpa info for all namespaces.
func Run(vpaLabels map[string]string) (Summary, error) {
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

	var summary Summary
	if len(vpas.Items) <= 0 {
		return summary, nil
	}
	for _, vpa := range vpas.Items {
		klog.V(8).Infof("Analyzing vpa: %v", vpa.ObjectMeta.Name)

		var deployment deploymentSummary
		deployment.DeploymentName = vpa.ObjectMeta.Name
		deployment.Namespace = vpa.ObjectMeta.Namespace
		if vpa.Status.Recommendation == nil {
			klog.V(2).Infof("Empty status on %v", deployment.DeploymentName)
			break
		}
		if len(vpa.Status.Recommendation.ContainerRecommendations) <= 0 {
			klog.V(2).Infof("No recommendations found in the %v vpa.", deployment.DeploymentName)
			break
		}
		for _, containerRecommendation := range vpa.Status.Recommendation.ContainerRecommendations {
			container := containerSummary{
				ContainerName: containerRecommendation.ContainerName,
				UpperBound:    containerRecommendation.UpperBound,
				LowerBound:    containerRecommendation.LowerBound,
			}
			deployment.Containers = append(deployment.Containers, container)
		}
		summary.Deployments = append(summary.Deployments, deployment)
	}

	return summary, nil
}
