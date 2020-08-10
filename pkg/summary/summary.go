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
	"k8s.io/apimachinery/pkg/util/sets"
	v1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2"
	"k8s.io/klog"

	"github.com/fairwindsops/goldilocks/pkg/utils"
)

const (
	namespaceAllNamespaces = ""
)

// Summary is for storing a summary of recommendation data by namespace/deployment/container
type Summary struct {
	Namespaces map[string]namespaceSummary
}

type namespaceSummary struct {
	Namespace   string                       `json:"namespace"`
	Deployments map[string]deploymentSummary `json:"deployments"`
}

type deploymentSummary struct {
	DeploymentName string                      `json:"deploymentName"`
	Containers     map[string]containerSummary `json:"containers"`
}

type containerSummary struct {
	ContainerName string `json:"containerName"`

	// recommendations
	LowerBound     corev1.ResourceList `json:"lowerBound"`
	UpperBound     corev1.ResourceList `json:"upperBound"`
	Target         corev1.ResourceList `json:"target"`
	UncappedTarget corev1.ResourceList `json:"uncappedTarget"`
	Limits         corev1.ResourceList `json:"limits"`
	Requests       corev1.ResourceList `json:"requests"`
}

// Summarizer represents a source of generating a summary of VPAs
type Summarizer struct {
	options

	// cached list of vpas
	vpas []v1beta2.VerticalPodAutoscaler
}

// NewSummarizer returns a Summarizer for all goldilocks managed VPAs in all Namespaces
func NewSummarizer(setters ...Option) *Summarizer {
	opts := defaultOptions()
	for _, setter := range setters {
		setter(opts)
	}

	return &Summarizer{
		options: *opts,
	}
}

// NewSummarizerForVPAs returns a Summarizer for a known list of VPAs
func NewSummarizerForVPAs(vpas []v1beta2.VerticalPodAutoscaler, setters ...Option) *Summarizer {
	summarizer := NewSummarizer(setters...)

	// set the cached vpas list directly
	summarizer.vpas = vpas

	return summarizer
}

// GetSummary returns a Summary of the Summarizer using its options
func (s Summarizer) GetSummary() (Summary, error) {
	summary := Summary{
		Namespaces: map[string]namespaceSummary{},
	}
	// cached vpas
	if s.vpas == nil {
		err := s.UpdateVPAs()
		if err != nil {
			return summary, err
		}
	}

	if len(s.vpas) <= 0 {
		return summary, nil
	}

	for _, vpa := range s.vpas {
		klog.V(8).Infof("Analyzing vpa: %v", vpa.Name)

		// get or create the namespaceSummary for this VPA's namespace
		namespace := vpa.Namespace
		var nsSummary namespaceSummary
		if val, ok := summary.Namespaces[namespace]; ok {
			nsSummary = val
		} else {
			nsSummary = namespaceSummary{
				Namespace:   namespace,
				Deployments: map[string]deploymentSummary{},
			}
			summary.Namespaces[namespace] = nsSummary
		}

		// VPA.Name := Deployment.Name, as that's how goldilocks works
		dSummary := deploymentSummary{
			DeploymentName: vpa.Name,
			Containers:     map[string]containerSummary{},
		}

		// find all deployments in the namespace for this VPA
		deployment, err := s.kubeClient.Client.AppsV1().Deployments(namespace).Get(dSummary.DeploymentName, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("Error retrieving deployment from API: %v", err)
		}

		if vpa.Status.Recommendation == nil {
			klog.V(2).Infof("Empty status on %v", dSummary.DeploymentName)
			continue
		}
		if len(vpa.Status.Recommendation.ContainerRecommendations) <= 0 {
			klog.V(2).Infof("No recommendations found in the %v vpa.", dSummary.DeploymentName)
			continue
		}

		// get the full set of excluded containers for this Deployment
		excludedContainers := sets.NewString().Union(s.excludedContainers)
		if val, exists := deployment.GetAnnotations()[utils.DeploymentExcludeContainersAnnotation]; exists {
			excludedContainers.Insert(strings.Split(val, ",")...)
		}

	CONTAINER_REC_LOOP:
		for _, containerRecommendation := range vpa.Status.Recommendation.ContainerRecommendations {
			if excludedContainers.Has(containerRecommendation.ContainerName) {
				klog.V(2).Infof("Excluding container Deployment/%s/%s", dSummary.DeploymentName, containerRecommendation.ContainerName)
				continue CONTAINER_REC_LOOP
			}

			var cSummary containerSummary
			for _, c := range deployment.Spec.Template.Spec.Containers {
				// find the matching container on the deployment
				if c.Name == containerRecommendation.ContainerName {
					cSummary = containerSummary{
						ContainerName:  containerRecommendation.ContainerName,
						UpperBound:     utils.FormatResourceList(containerRecommendation.UpperBound),
						LowerBound:     utils.FormatResourceList(containerRecommendation.LowerBound),
						Target:         utils.FormatResourceList(containerRecommendation.Target),
						UncappedTarget: utils.FormatResourceList(containerRecommendation.UncappedTarget),
						Limits:         utils.FormatResourceList(c.Resources.Limits),
						Requests:       utils.FormatResourceList(c.Resources.Requests),
					}
					klog.V(6).Infof("Resources for Deployment/%s/%s: Requests: %v Limits: %v", dSummary.DeploymentName, c.Name, cSummary.Requests, cSummary.Limits)
				}
			}

			dSummary.Containers[cSummary.ContainerName] = cSummary
		}

		// update summary maps
		nsSummary.Deployments[dSummary.DeploymentName] = dSummary
		summary.Namespaces[nsSummary.Namespace] = nsSummary
	}

	return summary, nil
}

// UpdateVPAs updates the list of VPAs that the summarizer uses
func (s *Summarizer) UpdateVPAs() error {
	nsLog := s.namespace
	if s.namespace == namespaceAllNamespaces {
		nsLog = "all namespaces"
	}
	klog.V(3).Infof("Looking for VPAs in %s with labels: %v", nsLog, s.vpaLabels)
	vpas, err := s.listVPAs()
	if err != nil {
		klog.Error(err.Error())
		return err
	}
	klog.V(10).Infof("Found vpas: %v", vpas)

	s.vpas = vpas
	return nil
}

// Run creates a summary of the vpa info for all namespaces.
func (s Summarizer) listVPAs() ([]v1beta2.VerticalPodAutoscaler, error) {
	vpaListOptions := getVPAListOptionsForLabels(s.vpaLabels)
	vpas, err := s.vpaClient.Client.AutoscalingV1beta2().VerticalPodAutoscalers(s.namespace).List(vpaListOptions)
	if err != nil {
		return nil, err
	}

	return vpas.Items, nil
}

func getVPAListOptionsForLabels(vpaLabels map[string]string) metav1.ListOptions {
	return metav1.ListOptions{
		LabelSelector: labels.Set(vpaLabels).String(),
	}
}
