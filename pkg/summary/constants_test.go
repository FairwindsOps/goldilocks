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
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"

	"github.com/fairwindsops/goldilocks/pkg/utils"
)

// These are all variables for testing. These resources, when applied to a cluster, should generate the testSummary at the end.

var updateMode = vpav1.UpdateModeOff

// lowerBound is used for all lower bounds in VPAs with recommendations
var lowerBound = v1.ResourceList{
	v1.ResourceCPU:    resource.MustParse("10m"),
	v1.ResourceMemory: resource.MustParse("10Mi"),
}

// upperBound is used for all upper bounds in VPAs with recommendations
var upperBound = v1.ResourceList{
	v1.ResourceCPU:    resource.MustParse("500m"),
	v1.ResourceMemory: resource.MustParse("500Mi"),
}

// targetResources is used for targets, requests, and limits on VPAs and Containers
var targetResources = v1.ResourceList{
	v1.ResourceCPU:    resource.MustParse("100m"),
	v1.ResourceMemory: resource.MustParse("100Mi"),
}

// Basic VPA and Deployment

var testVPABasic = &vpav1.VerticalPodAutoscaler{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-basic",
		Labels:    utils.VPALabels,
		Namespace: "testing",
	},
	Spec: vpav1.VerticalPodAutoscalerSpec{
		TargetRef: &autoscalingv1.CrossVersionObjectReference{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "test-basic",
		},
		UpdatePolicy: &vpav1.PodUpdatePolicy{
			UpdateMode: &updateMode,
		},
	},
	Status: vpav1.VerticalPodAutoscalerStatus{
		Conditions: []vpav1.VerticalPodAutoscalerCondition{
			{
				Message: "idk",
			},
		},
	},
}
var testDeploymentBasic = &appsv1.Deployment{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-basic",
		Namespace: "testing",
	},
}

// Not labelled for use by Goldilocks

var testVPANoLabels = &vpav1.VerticalPodAutoscaler{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-vpa-no-labels",
		Namespace: "testing",
	},
	Spec: vpav1.VerticalPodAutoscalerSpec{
		TargetRef: &autoscalingv1.CrossVersionObjectReference{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "test-vpa-no-labels",
		},
		UpdatePolicy: &vpav1.PodUpdatePolicy{
			UpdateMode: &updateMode,
		},
	},
}

// A complete VPA with recommendations

var testVPAWithReco = &vpav1.VerticalPodAutoscaler{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-vpa-with-reco",
		Namespace: "testing",
		Labels:    utils.VPALabels,
	},
	Spec: vpav1.VerticalPodAutoscalerSpec{
		TargetRef: &autoscalingv1.CrossVersionObjectReference{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "test-vpa-with-reco",
		},
		UpdatePolicy: &vpav1.PodUpdatePolicy{
			UpdateMode: &updateMode,
		},
	},
	Status: vpav1.VerticalPodAutoscalerStatus{
		Recommendation: &vpav1.RecommendedPodResources{
			ContainerRecommendations: []vpav1.RecommendedContainerResources{
				{
					ContainerName: "container",
					Target:        targetResources,
					UpperBound:    upperBound,
					LowerBound:    lowerBound,
				},
			},
		},
	},
}

var testDeploymentWithReco = &appsv1.Deployment{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-vpa-with-reco",
		Namespace: "testing",
	},
	Spec: appsv1.DeploymentSpec{
		Template: v1.PodTemplateSpec{
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name: "container",
						Resources: v1.ResourceRequirements{
							Limits:   targetResources,
							Requests: targetResources,
						},
					},
				},
			},
		},
	},
}

var testVPAWithUnmetReco = &vpav1.VerticalPodAutoscaler{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-vpa-with-unmet-reco",
		Namespace: "testing",
		Labels:    utils.VPALabels,
	},
	Spec: vpav1.VerticalPodAutoscalerSpec{
		TargetRef: &autoscalingv1.CrossVersionObjectReference{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "test-vpa-with-unmet-reco",
		},
		UpdatePolicy: &vpav1.PodUpdatePolicy{
			UpdateMode: &updateMode,
		},
	},
	Status: vpav1.VerticalPodAutoscalerStatus{
		Recommendation: &vpav1.RecommendedPodResources{
			ContainerRecommendations: []vpav1.RecommendedContainerResources{
				{
					ContainerName: "container",
					Target:        targetResources,
					UpperBound:    upperBound,
					LowerBound:    lowerBound,
				},
			},
		},
	},
}

var testDeploymentWithUnmetReco = &appsv1.Deployment{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-vpa-with-unmet-reco",
		Namespace: "testing",
	},
	Spec: appsv1.DeploymentSpec{
		Template: v1.PodTemplateSpec{
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name: "container",
						Resources: v1.ResourceRequirements{
							Limits:   lowerBound,
							Requests: lowerBound,
						},
					},
				},
			},
		},
	},
}

// The summary of these objects

var testSummary = Summary{
	Namespaces: map[string]namespaceSummary{
		"testing": {
			Namespace: "testing",
			Deployments: map[string]deploymentSummary{
				"test-basic": {
					DeploymentName: "test-basic",
					Containers:     map[string]containerSummary{},
				},
				"test-vpa-with-reco": {
					DeploymentName: "test-vpa-with-reco",
					Containers: map[string]containerSummary{
						"container": {
							ContainerName: "container",
							LowerBound:    lowerBound,
							UpperBound:    upperBound,
							Target:        targetResources,
							Limits:        targetResources,
							Requests:      targetResources,
						},
					},
				},
			},
		},
	},
	Filter: "all",
}

var testAnySummary = Summary{
	Namespaces: map[string]namespaceSummary{
		"testing": {
			Namespace: "testing",
			Deployments: map[string]deploymentSummary{
				"test-basic": {
					DeploymentName: "test-basic",
					Containers:     map[string]containerSummary{},
				},
			},
		},
	},
	Filter: "any",
}

var testGuaranteedSummary = Summary{
	Namespaces: map[string]namespaceSummary{
		"testing": {
			Namespace: "testing",
			Deployments: map[string]deploymentSummary{
				"test-basic": {
					DeploymentName: "test-basic",
					Containers:     map[string]containerSummary{},
				},
				"test-vpa-with-unmet-reco": {
					DeploymentName: "test-vpa-with-unmet-reco",
					Containers: map[string]containerSummary{
						"container": {
							ContainerName: "container",
							LowerBound:    lowerBound,
							UpperBound:    upperBound,
							Target:        targetResources,
							Limits:        lowerBound,
							Requests:      lowerBound,
						},
					},
				},
			},
		},
	},
	Filter: "guaranteed",
}
