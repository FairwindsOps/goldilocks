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
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
		Name:      "goldilocks-test-basic",
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

var testDeploymentBasicUnstructured = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind":       "Deployment",
		"apiVersion": "apps/v1",
		"metadata": map[string]interface{}{
			"name":      "test-basic",
			"namespace": "testing",
		},
		"spec": map[string]interface{}{},
	},
}

var testDeploymentBasicReplicaSetUnstructured = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind":       "ReplicaSet",
		"apiVersion": "apps/v1",
		"metadata": map[string]interface{}{
			"ownerReferences": []interface{}{
				map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"controller": true,
					"name":       "test-basic",
				},
			},
			"name":      "test-basic-0123456789",
			"namespace": "testing",
		},
		"spec": map[string]interface{}{},
	},
}

var testDeploymentBasicPodUnstructured = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind":       "Pod",
		"apiVersion": "v1",
		"metadata": map[string]interface{}{
			"ownerReferences": []interface{}{
				map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "ReplicaSet",
					"controller": true,
					"name":       "test-basic-0123456789",
				},
			},
			"name":      "test-basic-0123456789-01234",
			"namespace": "testing",
		},
		"spec": map[string]interface{}{},
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
		Name:      "goldilocks-test-vpa-with-reco",
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

var testDeploymentWithRecoUnstructured = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind":       "Deployment",
		"apiVersion": "apps/v1",
		"metadata": map[string]interface{}{
			"name":      "test-vpa-with-reco",
			"namespace": "testing",
		},
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{
							"name": "container",
							"resources": map[string]interface{}{
								"limits": map[string]interface{}{
									"cpu":    "100m",
									"memory": "100Mi",
								},
								"requests": map[string]interface{}{
									"cpu":    "100m",
									"memory": "100Mi",
								},
							},
						},
					},
				},
			},
		},
	},
}

var testDeploymentWithRecoReplicaSetUnstructured = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind":       "ReplicaSet",
		"apiVersion": "apps/v1",
		"metadata": map[string]interface{}{
			"ownerReferences": []interface{}{
				map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"controller": true,
					"name":       "test-vpa-with-reco",
				},
			},
			"name":      "test-vpa-with-reco-0123456789",
			"namespace": "testing",
		},
		"spec": map[string]interface{}{},
	},
}

var testDeploymentWithRecoPodUnstructured = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind":       "Pod",
		"apiVersion": "v1",
		"metadata": map[string]interface{}{
			"ownerReferences": []interface{}{
				map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "ReplicaSet",
					"controller": true,
					"name":       "test-vpa-with-reco-0123456789",
				},
			},
			"name":      "test-vpa-with-reco-0123456789-01234",
			"namespace": "testing",
		},
		"spec": map[string]interface{}{},
	},
}

// The summary of these objects

var testSummary = Summary{
	Namespaces: map[string]namespaceSummary{
		"testing": {
			Namespace: "testing",
			IsOnlyNamespace: true,
			Workloads: map[string]workloadSummary{
				"test-basic": {
					ControllerName: "test-basic",
					ControllerType: "Deployment",
					Containers:     map[string]ContainerSummary{},
				},
				"test-vpa-with-reco": {
					ControllerName: "test-vpa-with-reco",
					ControllerType: "Deployment",
					Containers: map[string]ContainerSummary{
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
}

// DaemonSet test and VPA

var testDaemonSettWithRecoUnstructured = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind":       "DaemonSet",
		"apiVersion": "apps/v1",
		"metadata": map[string]interface{}{
			"name":      "test-ds-with-reco",
			"namespace": "testing-daemonset",
		},
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{
							"name": "container",
							"resources": map[string]interface{}{
								"limits": map[string]interface{}{
									"cpu":    "100m",
									"memory": "100Mi",
								},
								"requests": map[string]interface{}{
									"cpu":    "100m",
									"memory": "100Mi",
								},
							},
						},
					},
				},
			},
		},
	},
}

var testDaemonSetWithRecoPodUnstructured = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind":       "Pod",
		"apiVersion": "v1",
		"metadata": map[string]interface{}{
			"ownerReferences": []interface{}{
				map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "DaemonSet",
					"controller": true,
					"name":       "test-ds-with-reco",
				},
			},
			"name":      "test-ds-with-reco-01234",
			"namespace": "testing-daemonset",
		},
		"spec": map[string]interface{}{},
	},
}

var testDaemonSetVPAWithReco = &vpav1.VerticalPodAutoscaler{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "goldilocks-test-ds-with-reco",
		Namespace: "testing-daemonset",
		Labels:    utils.VPALabels,
	},
	Spec: vpav1.VerticalPodAutoscalerSpec{
		TargetRef: &autoscalingv1.CrossVersionObjectReference{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
			Name:       "test-ds-with-reco",
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

// The summary of the daemonset

var testSummaryDaemonSet = Summary{
	Namespaces: map[string]namespaceSummary{
		"testing-daemonset": {
			Namespace: "testing-daemonset",
			IsOnlyNamespace: true,
			Workloads: map[string]workloadSummary{
				"test-ds-with-reco": {
					ControllerName: "test-ds-with-reco",
					ControllerType: "DaemonSet",
					Containers: map[string]ContainerSummary{
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
}
