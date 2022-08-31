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

	controllerUtils "github.com/fairwindsops/controller-utils/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/klog/v2"

	"github.com/fairwindsops/goldilocks/pkg/utils"
)

const (
	namespaceAllNamespaces = ""
)

// Summary is for storing a summary of recommendation data by namespace/controller type/container
type Summary struct {
	Namespaces map[string]namespaceSummary
}

type namespaceSummary struct {
	Namespace string                     `json:"namespace"`
	Workloads map[string]workloadSummary `json:"workloads"`
	BasePath  string
}

type workloadSummary struct {
	ControllerName string                      `json:"controllerName"`
	ControllerType string                      `json:"controllerType"`
	Containers     map[string]containerSummary `json:"containers"`
	BasePath       string
}

type containerSummary struct {
	ContainerName string `json:"containerName"`

	// recommendations
	LowerBound      corev1.ResourceList `json:"lowerBound"`
	UpperBound      corev1.ResourceList `json:"upperBound"`
	Target          corev1.ResourceList `json:"target"`
	UncappedTarget  corev1.ResourceList `json:"uncappedTarget"`
	Limits          corev1.ResourceList `json:"limits"`
	Requests        corev1.ResourceList `json:"requests"`
	BasePath        string
	TopNumber       float64
	LowerNumber1    float64
	LowerNumber2    float64
	LowerNumberInt1 int
	LowerNumberInt2 int
}

// Summarizer represents a source of generating a summary of VPAs
type Summarizer struct {
	options

	// cached list of vpas
	vpas []vpav1.VerticalPodAutoscaler

	// cached map of vpa name -> workload
	workloadForVPANamed map[string]*controllerUtils.Workload
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
			Namespace: s.namespace,
			Workloads: map[string]workloadSummary{},
		}
	}

	// cached vpas and workloads
	if s.vpas == nil || s.workloadForVPANamed == nil {
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
				Namespace: namespace,
				Workloads: map[string]workloadSummary{},
			}
			summary.Namespaces[namespace] = nsSummary
		}

		wSummary := workloadSummary{
			ControllerName: vpa.Spec.TargetRef.Name,
			ControllerType: vpa.Spec.TargetRef.Kind,
			Containers:     map[string]containerSummary{},
		}

		workload, ok := s.workloadForVPANamed[vpa.Name]
		if !ok {
			klog.Errorf("no matching Workloads found for VPA/%s", vpa.Name)
			continue
		}

		if vpa.Status.Recommendation == nil {
			klog.V(2).Infof("Empty status on %v", wSummary.ControllerName)
			nsSummary.Workloads[wSummary.ControllerName] = wSummary
			summary.Namespaces[nsSummary.Namespace] = nsSummary
			continue
		}
		if len(vpa.Status.Recommendation.ContainerRecommendations) <= 0 {
			klog.V(2).Infof("No container recommendations found in the %v vpa.", wSummary.ControllerName)
			nsSummary.Workloads[wSummary.ControllerName] = wSummary
			summary.Namespaces[nsSummary.Namespace] = nsSummary
			continue
		}

		// get the full set of excluded containers for this workload
		excludedContainers := sets.NewString().Union(s.excludedContainers)
		if val, exists := workload.TopController.GetAnnotations()[utils.WorkloadExcludeContainersAnnotation]; exists {
			excludedContainers.Insert(strings.Split(val, ",")...)
		}

	CONTAINER_REC_LOOP:
		for _, containerRecommendation := range vpa.Status.Recommendation.ContainerRecommendations {
			if excludedContainers.Has(containerRecommendation.ContainerName) {
				klog.V(2).Infof("Excluding container %s/%s/%s", wSummary.ControllerType, wSummary.ControllerName, containerRecommendation.ContainerName)
				continue CONTAINER_REC_LOOP
			}

			var cSummary containerSummary
			workloadPodSpecUnstructured, workloadPodSpecFound, err := unstructured.NestedMap(workload.TopController.UnstructuredContent(), "spec", "template", "spec")
			if err != nil {
				klog.Errorf("unable to parse spec.template.spec from unstructured workload. Namespace: '%s', Kind: '%s', Name: '%s'", workload.TopController.GetNamespace(), workload.TopController.GetKind(), workload.TopController.GetName())
				continue CONTAINER_REC_LOOP
			}
			if !workloadPodSpecFound {
				klog.Errorf("no spec.template.spec field from unstructured workload. Namespace: '%s', Kind: '%s', Name: '%s'", workload.TopController.GetNamespace(), workload.TopController.GetKind(), workload.TopController.GetName())
				continue CONTAINER_REC_LOOP
			}

			var workloadPodSpec corev1.PodSpec
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(workloadPodSpecUnstructured, &workloadPodSpec)
			if err != nil {
				klog.Errorf("unable to convert unstructured pod spec to PodSpec struct. Namespace: '%s', Kind: '%s', Name: '%s'", workload.TopController.GetNamespace(), workload.TopController.GetKind(), workload.TopController.GetName())
				continue CONTAINER_REC_LOOP
			}

			for _, c := range workloadPodSpec.Containers {
				// find the matching container on the workload
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
					klog.V(6).Infof("Resources for %s/%s/%s: Requests: %v Limits: %v", wSummary.ControllerType, wSummary.ControllerName, c.Name, cSummary.Requests, cSummary.Limits)
					wSummary.Containers[cSummary.ContainerName] = cSummary
					continue CONTAINER_REC_LOOP
				}
			}
		}
		// update summary maps
		nsSummary.Workloads[wSummary.ControllerName] = wSummary
		summary.Namespaces[nsSummary.Namespace] = nsSummary
	}

	return summary, nil
}

// Update the set of VPAs and Workloads that the Summarizer uses for creating a summary
func (s *Summarizer) Update() error {
	err := s.updateVPAs()
	if err != nil {
		klog.Error(err.Error())
		return err
	}

	err = s.updateWorkloads()
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

func (s *Summarizer) updateWorkloads() error {
	nsLog := s.namespace
	if s.namespace == namespaceAllNamespaces {
		nsLog = "all namespaces"
	}
	klog.V(3).Infof("Looking for Workloads in %s", nsLog)
	workloads, err := s.listWorkloads()
	if err != nil {
		return err
	}
	klog.V(10).Infof("Found workloads in namespace '%s': %v", s.namespace, workloads)

	// map goldilocks-workload.name -> &controllerUtils.Workload{} for easy vpa lookup.
	s.workloadForVPANamed = map[string]*controllerUtils.Workload{}
	for _, w := range workloads {
		for _, v := range s.vpas {
			w := w
			if vpaMatchesWorkload(v, w) {
				vpaName := v.Name
				s.workloadForVPANamed[vpaName] = &w
			}
		}
	}

	return nil
}

// vpaMatchesWorkload returns true if the VPA's target matches the workload
func vpaMatchesWorkload(v vpav1.VerticalPodAutoscaler, w controllerUtils.Workload) bool {
	// check if the VPA's target matches the workload's target
	if v.Spec.TargetRef.Kind != w.TopController.GetKind() {
		return false
	}
	if v.Spec.TargetRef.Name != w.TopController.GetName() {
		return false
	}
	if v.Spec.TargetRef.APIVersion != w.TopController.GetAPIVersion() {
		return false
	}
	return true
}

func (s Summarizer) listWorkloads() ([]controllerUtils.Workload, error) {
	workloads, err := controllerUtils.GetAllTopControllers(context.TODO(), s.dynamicClient.Client, s.dynamicClient.RESTMapper, s.namespace)
	if err != nil {
		return nil, err
	}

	return workloads, nil
}
