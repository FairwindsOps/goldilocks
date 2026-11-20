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
	"maps"
	"slices"
	"testing"

	"github.com/fairwindsops/goldilocks/pkg/kube"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
)

func Test_Summarizer(t *testing.T) {
	kubeClientVPA := kube.GetMockVPAClient()
	kubeClient := kube.GetMockClient()
	dynamicClient := kube.GetMockDynamicClient()
	controllerUtilsClient := kube.GetMockControllerUtilsClient(dynamicClient)

	summarizer := NewSummarizer()
	summarizer.kubeClient = kubeClient
	summarizer.vpaClient = kubeClientVPA
	summarizer.dynamicClient = dynamicClient
	summarizer.controllerUtilsClient = controllerUtilsClient

	// _, _ = dynamicClient.Client.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}).Create(context.TODO(), nsLabeledTrueUnstructured, metav1.CreateOptions{})
	_, err := dynamicClient.Client.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}).Namespace("testing").Create(context.TODO(), testDeploymentBasicUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = dynamicClient.Client.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"}).Namespace("testing").Create(context.TODO(), testDeploymentBasicReplicaSetUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = dynamicClient.Client.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}).Namespace("testing").Create(context.TODO(), testDeploymentBasicPodUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, errOk := kubeClientVPA.Client.AutoscalingV1().VerticalPodAutoscalers("testing").Create(context.TODO(), testVPABasic, metav1.CreateOptions{})
	assert.NoError(t, errOk)

	_, errOk2 := kubeClientVPA.Client.AutoscalingV1().VerticalPodAutoscalers("testing").Create(context.TODO(), testVPANoLabels, metav1.CreateOptions{})
	assert.NoError(t, errOk2)

	_, err = dynamicClient.Client.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}).Namespace("testing").Create(context.TODO(), testDeploymentWithRecoUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = dynamicClient.Client.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"}).Namespace("testing").Create(context.TODO(), testDeploymentWithRecoReplicaSetUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = dynamicClient.Client.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}).Namespace("testing").Create(context.TODO(), testDeploymentWithRecoPodUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, errOk3 := kubeClientVPA.Client.AutoscalingV1().VerticalPodAutoscalers("testing").Create(context.TODO(), testVPAWithReco, metav1.CreateOptions{})
	assert.NoError(t, errOk3)

	got, err := summarizer.GetSummary()
	assert.NoError(t, err)

	assert.EqualValues(t, testSummary, got)
}

func Test_Summarizer_Daemonset(t *testing.T) {
	kubeClientVPA := kube.GetMockVPAClient()
	kubeClient := kube.GetMockClient()
	dynamicClient := kube.GetMockDynamicClient()
	controllerUtilsClient := kube.GetMockControllerUtilsClient(dynamicClient)

	summarizer := NewSummarizer()
	summarizer.kubeClient = kubeClient
	summarizer.vpaClient = kubeClientVPA
	summarizer.dynamicClient = dynamicClient
	summarizer.controllerUtilsClient = controllerUtilsClient

	// _, _ = dynamicClient.Client.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}).Create(context.TODO(), nsLabeledTrueUnstructured, metav1.CreateOptions{})
	_, err := dynamicClient.Client.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}).Namespace("testing-daemonset").Create(context.TODO(), testDaemonSettWithRecoUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = dynamicClient.Client.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}).Namespace("testing-daemonset").Create(context.TODO(), testDaemonSetWithRecoPodUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, errOk := kubeClientVPA.Client.AutoscalingV1().VerticalPodAutoscalers("testing-daemonset").Create(context.TODO(), testDaemonSetVPAWithReco, metav1.CreateOptions{})
	assert.NoError(t, errOk)

	got, err := summarizer.GetSummary()
	assert.NoError(t, err)

	assert.EqualValues(t, testSummaryDaemonSet, got)
}

func Test_Summarizer_VPAWithoutTargetRef(t *testing.T) {
	kubeClientVPA := kube.GetMockVPAClient()
	kubeClient := kube.GetMockClient()
	dynamicClient := kube.GetMockDynamicClient()
	controllerUtilsClient := kube.GetMockControllerUtilsClient(dynamicClient)

	summarizer := NewSummarizer()
	summarizer.kubeClient = kubeClient
	summarizer.vpaClient = kubeClientVPA
	summarizer.dynamicClient = dynamicClient
	summarizer.controllerUtilsClient = controllerUtilsClient

	_, err := dynamicClient.Client.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}).Namespace("testing").Create(context.TODO(), testDeploymentBasicUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = dynamicClient.Client.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"}).Namespace("testing").Create(context.TODO(), testDeploymentBasicReplicaSetUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = dynamicClient.Client.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}).Namespace("testing").Create(context.TODO(), testDeploymentBasicPodUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = kubeClientVPA.Client.AutoscalingV1().VerticalPodAutoscalers("testing").Create(context.TODO(), testVPANoTargetRef, metav1.CreateOptions{})
	assert.NoError(t, err)

	// a VPA with no targetRef should be skipped, not bring the summary down
	got, err := summarizer.GetSummary()
	assert.NoError(t, err)
	assert.Empty(t, got.Namespaces)
}

func Test_Summarizer_ExcludedContainers(t *testing.T) {
	tests := []struct {
		name string
		// value of the exclude-containers annotation on the workload, not set when empty
		annotation string
		// containers excluded for every workload, like the --exclude-containers argument
		excluded []string
		want     []string
	}{
		{
			name: "nothing excluded",
			want: []string{"app", "istio-proxy", "linkerd-proxy"},
		},
		{
			name:       "annotation on the workload",
			annotation: "istio-proxy,linkerd-proxy",
			want:       []string{"app"},
		},
		{
			name:       "annotation with spaces after the commas",
			annotation: "istio-proxy, linkerd-proxy",
			want:       []string{"app"},
		},
		{
			name:       "annotation combined with the excluded containers option",
			annotation: "istio-proxy",
			excluded:   []string{"linkerd-proxy"},
			want:       []string{"app"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			kubeClientVPA := kube.GetMockVPAClient()
			kubeClient := kube.GetMockClient()
			dynamicClient := kube.GetMockDynamicClient()
			controllerUtilsClient := kube.GetMockControllerUtilsClient(dynamicClient)

			summarizer := NewSummarizer(ExcludeContainers(sets.New(tc.excluded...)))
			summarizer.kubeClient = kubeClient
			summarizer.vpaClient = kubeClientVPA
			summarizer.dynamicClient = dynamicClient
			summarizer.controllerUtilsClient = controllerUtilsClient

			_, err := dynamicClient.Client.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}).Namespace("testing-exclude").Create(context.TODO(), testDaemonSetExcludeContainers(tc.annotation), metav1.CreateOptions{})
			assert.NoError(t, err)
			_, err = dynamicClient.Client.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}).Namespace("testing-exclude").Create(context.TODO(), testDaemonSetExcludePodUnstructured, metav1.CreateOptions{})
			assert.NoError(t, err)
			_, err = kubeClientVPA.Client.AutoscalingV1().VerticalPodAutoscalers("testing-exclude").Create(context.TODO(), testDaemonSetExcludeVPA(), metav1.CreateOptions{})
			assert.NoError(t, err)

			got, err := summarizer.GetSummary()
			assert.NoError(t, err)

			containers := slices.Sorted(maps.Keys(got.Namespaces["testing-exclude"].Workloads["test-ds-exclude"].Containers))
			assert.Equal(t, tc.want, containers)
		})
	}
}
