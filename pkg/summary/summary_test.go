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
	"github.com/fairwindsops/goldilocks/pkg/utils"
	"github.com/stretchr/testify/assert"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
)

func TestSummarizer(t *testing.T) {
	kubeClientVPA := kube.GetMockVPAClient()
	kubeClient := kube.GetMockClient()

	summarizer := NewSummarizer()
	summarizer.kubeClient = kubeClient
	summarizer.vpaClient = kubeClientVPA

	updateMode := vpav1.UpdateModeOff
	var testVPA = &vpav1.VerticalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vpa",
			Labels:    utils.VPALabels,
			Namespace: "testing",
		},
		Spec: vpav1.VerticalPodAutoscalerSpec{
			TargetRef: &autoscalingv1.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       "test-vpa",
			},
			UpdatePolicy: &vpav1.PodUpdatePolicy{
				UpdateMode: &updateMode,
			},
		},
	}
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

	_, errOk := kubeClientVPA.Client.AutoscalingV1().VerticalPodAutoscalers("testing").Create(context.TODO(), testVPA, metav1.CreateOptions{})
	assert.NoError(t, errOk)

	_, errOk2 := kubeClientVPA.Client.AutoscalingV1().VerticalPodAutoscalers("testing").Create(context.TODO(), testVPANoLabels, metav1.CreateOptions{})
	assert.NoError(t, errOk2)

	var summary = Summary{
		Namespaces: map[string]namespaceSummary{
			"testing": namespaceSummary{
				Namespace:   "testing",
				Deployments: map[string]deploymentSummary{},
			},
		},
	}

	got, err := summarizer.GetSummary()
	assert.NoError(t, err)

	assert.EqualValues(t, summary, got)
}
