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

	"github.com/fairwindsops/goldilocks/pkg/kube"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestSummarizer(t *testing.T) {
	kubeClientVPA := kube.GetMockVPAClient()
	// kubeClient := kube.GetMockClient()
	dynamicClient := kube.GetMockDynamicClient()

	summarizer := NewSummarizer()
	// summarizer.kubeClient = kubeClient
	summarizer.vpaClient = kubeClientVPA
	summarizer.dynamicClient = dynamicClient

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
