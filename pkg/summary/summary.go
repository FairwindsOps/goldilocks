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
	"encoding/json"
	"fmt"

	"k8s.io/klog"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/fairwindsops/vpa-analysis/pkg/kube"
)

type containerSummary struct {
	LowerBound    v1.ResourceList `json:"lowerBound"`
	UpperBound    v1.ResourceList `json:"upperBound"`
	ContainerName string          `json:"containerName"`
}

type deploymentSummary struct {
	Containers     []containerSummary `json:"containers"`
	DeploymentName string             `json:"deploymentName"`
}

// Summary struct is for storing a summary of recommendation data.
type Summary struct {
	Deployments []deploymentSummary `json:"deployments"`
}

// Run creates a summary of the vpa info
func Run(namespace string, kubeconfig *string, vpaLabels map[string]string) {
	klog.V(3).Infof("Gathering info for summary from namespace: %s", namespace)
	klog.V(3).Infof("Using Kubeconfig: %s", *kubeconfig)
	klog.V(3).Infof("Looking for vpa with labels: %v", vpaLabels)

	kubeClientVPA := kube.GetVPAInstance()

	vpaListOptions := metav1.ListOptions{
		LabelSelector: labels.Set(vpaLabels).String(),
	}

	vpas, err := kubeClientVPA.Client.AutoscalingV1beta2().VerticalPodAutoscalers(namespace).List(vpaListOptions)
	if err != nil {
		klog.Fatal(err.Error())
	}
	klog.V(10).Infof("Found vpas: %v", vpas)

	if len(vpas.Items) <= 0 {
		klog.Fatalf("No vpas were found in the %s namespace.", namespace)
	}
	var summary Summary
	for _, vpa := range vpas.Items {
		klog.V(8).Infof("Analyzing vpa: %v", vpa.ObjectMeta.Name)
		// TODO: This will segfault if it is run before recommendations are generated.  Need to catch that.
		klog.V(10).Info(vpa.Status.Recommendation.ContainerRecommendations)

		var deployment deploymentSummary
		deployment.DeploymentName = vpa.ObjectMeta.Name
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
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		klog.Fatalf("Error marshalling JSON: %v", err)
	}
	fmt.Println(string(summaryJSON))
}
