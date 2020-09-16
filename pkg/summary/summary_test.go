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
)

func TestSummarizer(t *testing.T) {
	kubeClientVPA := kube.GetMockVPAClient()
	kubeClient := kube.GetMockClient()

	summarizer := NewSummarizer()
	summarizer.kubeClient = kubeClient
	summarizer.vpaClient = kubeClientVPA

	kubeClient.Client.AppsV1().Deployments("testing").Create(context.TODO(), testDeploymentBasic, metav1.CreateOptions{})
	_, errOk := kubeClientVPA.Client.AutoscalingV1().VerticalPodAutoscalers("testing").Create(context.TODO(), testVPABasic, metav1.CreateOptions{})
	assert.NoError(t, errOk)

	_, errOk2 := kubeClientVPA.Client.AutoscalingV1().VerticalPodAutoscalers("testing").Create(context.TODO(), testVPANoLabels, metav1.CreateOptions{})
	assert.NoError(t, errOk2)

	kubeClient.Client.AppsV1().Deployments("testing").Create(context.TODO(), testDeploymentWithReco, metav1.CreateOptions{})
	_, errOk3 := kubeClientVPA.Client.AutoscalingV1().VerticalPodAutoscalers("testing").Create(context.TODO(), testVPAWithReco, metav1.CreateOptions{})
	assert.NoError(t, errOk3)

	got, err := summarizer.GetSummary()
	assert.NoError(t, err)

	assert.EqualValues(t, testSummary, got)
}
