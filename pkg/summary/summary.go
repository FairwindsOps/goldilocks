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
	"context"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
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
	vpas []vpav1.VerticalPodAutoscaler

	// cached map of deploy/vpa name -> deployment
	deploymentForVPANamed map[string]*appsv1.Deployment
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
func NewSummarizerForVPAs(vpas []vpav1.VerticalPodAutoscaler, setters ...Option) *Summarizer {
	summarizer := NewSummarizer(setters...)

	// set the cached vpas list directly
	summarizer.vpas = vpas

	return summarizer
}

// GetSummary returns a Summary of the Summarizer using its options
func (s Summarizer) GetSummary() (Summary, error) {
	// blank summary
	summary := Summary{
		Namespaces: map[string]namespaceSummary{},
	}

	// if the summarizer is filtering for a single namespace,
	// then add that namespace by default to the blank summary
	if s.namespace != namespaceAllNamespaces {
		summary.Namespaces[s.namespace] = namespaceSummary{
			Namespace:   s.namespace,
			Deployments: map[string]deploymentSummary{},
		}
	}

	// cached vpas and deployments
	if s.vpas == nil || s.deploymentForVPANamed == nil {
		err := s.Update()
		if err != nil {
			return summary, err
		}
	}

	// nothing to summarize
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

		deployment, ok := s.deploymentForVPANamed[vpa.Name]
		if !ok {
			klog.Errorf("no matching Deployment found for VPA/%s", vpa.Name)
			continue
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
					dSummary.Containers[cSummary.ContainerName] = cSummary
					continue CONTAINER_REC_LOOP
				}
			}
		}

		// update summary maps
		nsSummary.Deployments[dSummary.DeploymentName] = dSummary
		summary.Namespaces[nsSummary.Namespace] = nsSummary
	}

	return summary, nil
}

// Update the set of VPAs and Deployments that the Summarizer uses for creating a summary
func (s *Summarizer) Update() error {
	err := s.updateVPAs()
	if err != nil {
		klog.Error(err.Error())
		return err
	}

	err = s.updateDeployments()
	if err != nil {
		klog.Error(err.Error())
		return err
	}

	return nil
}

func (s *Summarizer) updateVPAs() error {
	nsLog := s.namespace
	if s.namespace == namespaceAllNamespaces {
		nsLog = "all namespaces"
	}
	klog.V(3).Infof("Looking for VPAs in %s with labels: %v", nsLog, s.vpaLabels)
	vpas, err := s.listVPAs(getVPAListOptionsForLabels(s.vpaLabels))
	if err != nil {
		return err
	}
	klog.V(10).Infof("Found vpas: %v", vpas)

	s.vpas = vpas
	return nil
}

func (s Summarizer) listVPAs(listOptions metav1.ListOptions) ([]vpav1.VerticalPodAutoscaler, error) {
	vpas, err := s.vpaClient.Client.AutoscalingV1().VerticalPodAutoscalers(s.namespace).List(context.TODO(), listOptions)
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

func (s *Summarizer) updateDeployments() error {
	nsLog := s.namespace
	if s.namespace == namespaceAllNamespaces {
		nsLog = "all namespaces"
	}
	klog.V(3).Infof("Looking for Deployments in %s", nsLog)
	deployments, err := s.listDeployments(metav1.ListOptions{})
	if err != nil {
		return err
	}
	klog.V(10).Infof("Found deployments: %v", deployments)

	// map the deployment.name -> &deployment for easy vpa lookup (since vpa.Name == deployment.Name for matching vpas/deployments)
	s.deploymentForVPANamed = map[string]*appsv1.Deployment{}
	for _, d := range deployments {
		d := d
		s.deploymentForVPANamed[d.Name] = &d
	}

	return nil
}

func (s Summarizer) listDeployments(listOptions metav1.ListOptions) ([]appsv1.Deployment, error) {
	deployments, err := s.kubeClient.Client.AppsV1().Deployments(s.namespace).List(context.TODO(), listOptions)
	if err != nil {
		return nil, err
	}

	return deployments.Items, nil
}
