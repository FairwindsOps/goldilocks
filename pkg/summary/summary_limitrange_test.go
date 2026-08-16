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
	"testing"

	"github.com/stretchr/testify/assert"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"

	"github.com/fairwindsops/goldilocks/pkg/kube"
	"github.com/fairwindsops/goldilocks/pkg/utils"
)

// Test_Summarizer_LimitRange exercises the full GetSummary() pipeline (not just the
// resolveEffectiveResources unit) for a workload whose container sets no resources of its own,
// in a namespace that has a LimitRange with both Default (memory) and DefaultRequest (cpu) set.
// This is the end-to-end version of issue #315: previously, this container would have shown
// "Not Set" for every value; it should now show the values a real Pod created from this
// workload would actually receive.
func Test_Summarizer_LimitRange(t *testing.T) {
	const namespace = "testing-limitrange"

	kubeClientVPA := kube.GetMockVPAClient()
	kubeClient := kube.GetMockClient()
	dynamicClient := kube.GetMockDynamicClient()
	controllerUtilsClient := kube.GetMockControllerUtilsClient(dynamicClient)

	summarizer := NewSummarizer()
	summarizer.kubeClient = kubeClient
	summarizer.vpaClient = kubeClientVPA
	summarizer.dynamicClient = dynamicClient
	summarizer.controllerUtilsClient = controllerUtilsClient

	// A namespace-wide LimitRange with Default memory (limit) and DefaultRequest cpu (request).
	_, err := kubeClient.Client.CoreV1().LimitRanges(namespace).Create(context.TODO(), &corev1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "limits",
			Namespace: namespace,
		},
		Spec: corev1.LimitRangeSpec{
			Limits: []corev1.LimitRangeItem{
				{
					Type: corev1.LimitTypeContainer,
					Default: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("512Mi"),
					},
					DefaultRequest: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("100m"),
					},
				},
			},
		},
	}, metav1.CreateOptions{})
	assert.NoError(t, err)

	deployment := &unstructured.Unstructured{
		Object: map[string]any{
			"kind":       "Deployment",
			"apiVersion": "apps/v1",
			"metadata": map[string]any{
				"name":      "test-limitrange-demo",
				"namespace": namespace,
			},
			"spec": map[string]any{
				"template": map[string]any{
					"spec": map[string]any{
						"containers": []any{
							map[string]any{
								"name": "container",
								// no resources set at all -- everything should come from the
								// namespace LimitRange (or remain "Not Set" for cpu limit,
								// since the LimitRange has no Default for cpu).
							},
						},
					},
				},
			},
		},
	}
	replicaSet := &unstructured.Unstructured{
		Object: map[string]any{
			"kind":       "ReplicaSet",
			"apiVersion": "apps/v1",
			"metadata": map[string]any{
				"ownerReferences": []any{
					map[string]any{
						"apiVersion": "apps/v1",
						"kind":       "Deployment",
						"controller": true,
						"name":       "test-limitrange-demo",
					},
				},
				"name":      "test-limitrange-demo-0123456789",
				"namespace": namespace,
			},
			"spec": map[string]any{},
		},
	}
	pod := &unstructured.Unstructured{
		Object: map[string]any{
			"kind":       "Pod",
			"apiVersion": "v1",
			"metadata": map[string]any{
				"ownerReferences": []any{
					map[string]any{
						"apiVersion": "apps/v1",
						"kind":       "ReplicaSet",
						"controller": true,
						"name":       "test-limitrange-demo-0123456789",
					},
				},
				"name":      "test-limitrange-demo-0123456789-01234",
				"namespace": namespace,
			},
			"spec": map[string]any{},
		},
	}

	_, err = dynamicClient.Client.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}).Namespace(namespace).Create(context.TODO(), deployment, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = dynamicClient.Client.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"}).Namespace(namespace).Create(context.TODO(), replicaSet, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = dynamicClient.Client.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}).Namespace(namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
	assert.NoError(t, err)

	vpa := &vpav1.VerticalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "goldilocks-test-limitrange-demo",
			Namespace: namespace,
			Labels:    utils.VPALabels,
		},
		Spec: vpav1.VerticalPodAutoscalerSpec{
			TargetRef: &autoscalingv1.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       "test-limitrange-demo",
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
	_, err = kubeClientVPA.Client.AutoscalingV1().VerticalPodAutoscalers(namespace).Create(context.TODO(), vpa, metav1.CreateOptions{})
	assert.NoError(t, err)

	got, err := summarizer.GetSummary()
	assert.NoError(t, err)

	nsSummary, ok := got.Namespaces[namespace]
	assert.True(t, ok, "expected a summary for namespace %q", namespace)

	wSummary, ok := nsSummary.Workloads["test-limitrange-demo"]
	assert.True(t, ok, "expected a workload summary for test-limitrange-demo")

	cSummary, ok := wSummary.Containers["container"]
	assert.True(t, ok, "expected a container summary for 'container'")

	// memory limit came from the LimitRange's Default
	assert.True(t, quantitiesEqual(corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("512Mi")},
		corev1.ResourceList{corev1.ResourceMemory: cSummary.Limits[corev1.ResourceMemory]}))
	assert.True(t, cSummary.LimitsFromLimitRange[corev1.ResourceMemory])

	// cpu request came from the LimitRange's DefaultRequest
	assert.True(t, quantitiesEqual(corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m")},
		corev1.ResourceList{corev1.ResourceCPU: cSummary.Requests[corev1.ResourceCPU]}))
	assert.True(t, cSummary.RequestsFromLimitRange[corev1.ResourceCPU])

	// cpu limit has no Default in the LimitRange and no explicit value -- stays unset ("Not Set")
	cpuLimit := cSummary.Limits[corev1.ResourceCPU]
	assert.True(t, cpuLimit.IsZero())
	assert.False(t, cSummary.LimitsFromLimitRange[corev1.ResourceCPU])

	// memory request has no DefaultRequest and the container has no explicit memory limit
	// either (so there's nothing to copy into the request) -- stays unset ("Not Set")
	memRequest := cSummary.Requests[corev1.ResourceMemory]
	assert.True(t, memRequest.IsZero())
	assert.False(t, cSummary.RequestsFromLimitRange[corev1.ResourceMemory])
}
