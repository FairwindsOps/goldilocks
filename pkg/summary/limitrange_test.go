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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fairwindsops/goldilocks/pkg/kube"
)

func Test_resolveEffectiveResources(t *testing.T) {
	containerItem := func(def, defReq corev1.ResourceList) corev1.LimitRangeItem {
		return corev1.LimitRangeItem{
			Type:           corev1.LimitTypeContainer,
			Default:        def,
			DefaultRequest: defReq,
		}
	}

	tests := []struct {
		name                       string
		requests                   corev1.ResourceList
		limits                     corev1.ResourceList
		limitRangeItems            []corev1.LimitRangeItem
		wantRequests               corev1.ResourceList
		wantLimits                 corev1.ResourceList
		wantRequestsFromLimitRange map[corev1.ResourceName]bool
		wantLimitsFromLimitRange   map[corev1.ResourceName]bool
	}{
		{
			name:            "no LimitRange in namespace is a no-op",
			requests:        corev1.ResourceList{},
			limits:          corev1.ResourceList{},
			limitRangeItems: nil,
			wantRequests:    corev1.ResourceList{},
			wantLimits:      corev1.ResourceList{},
		},
		{
			name:     "no LimitRange in namespace leaves explicit values untouched",
			requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m")},
			limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("200m")},
			wantRequests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("100m"),
			},
			wantLimits: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("200m"),
			},
		},
		{
			name:     "only Default set: unset container fills in Limit but Request stays unset",
			requests: corev1.ResourceList{},
			limits:   corev1.ResourceList{},
			limitRangeItems: []corev1.LimitRangeItem{
				containerItem(corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("512Mi")}, nil),
			},
			wantRequests: corev1.ResourceList{},
			wantLimits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi"),
			},
			wantLimitsFromLimitRange: map[corev1.ResourceName]bool{corev1.ResourceMemory: true},
		},
		{
			name:     "only DefaultRequest set: unset container fills in Request but Limit stays unset",
			requests: corev1.ResourceList{},
			limits:   corev1.ResourceList{},
			limitRangeItems: []corev1.LimitRangeItem{
				containerItem(nil, corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("256Mi")}),
			},
			wantRequests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("256Mi"),
			},
			wantLimits:                 corev1.ResourceList{},
			wantRequestsFromLimitRange: map[corev1.ResourceName]bool{corev1.ResourceMemory: true},
		},
		{
			name:     "both Default and DefaultRequest set on an unset container",
			requests: corev1.ResourceList{},
			limits:   corev1.ResourceList{},
			limitRangeItems: []corev1.LimitRangeItem{
				containerItem(
					corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("512Mi")},
					corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("256Mi")},
				),
			},
			wantRequests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("256Mi"),
			},
			wantLimits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi"),
			},
			wantRequestsFromLimitRange: map[corev1.ResourceName]bool{corev1.ResourceMemory: true},
			wantLimitsFromLimitRange:   map[corev1.ResourceName]bool{corev1.ResourceMemory: true},
		},
		{
			// This is the trickiest edge case: a container that explicitly sets ONLY a Limit
			// (no Request), in a namespace whose LimitRange has both Default and
			// DefaultRequest. Real Kubernetes applies its "request defaults to explicit limit"
			// Pod-level defaulting (SetDefaults_Pod) BEFORE LimitRanger ever runs, so the
			// request becomes the container's own explicit limit value -- NOT the namespace's
			// DefaultRequest. See https://kubernetes.io/docs/tasks/administer-cluster/manage-resources/memory-default-namespace/#if-you-specify-a-container-s-limit-but-not-its-request
			name:     "explicit limit with no request ignores DefaultRequest and copies the explicit limit",
			requests: corev1.ResourceList{},
			limits:   corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("1Gi")},
			limitRangeItems: []corev1.LimitRangeItem{
				containerItem(
					corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("512Mi")},
					corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("256Mi")},
				),
			},
			wantRequests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
			wantLimits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
			// Neither value came from the LimitRange: the limit is the container's own, and the
			// request is copied from that explicit limit, not from DefaultRequest.
		},
		{
			// Mirror sanity check: an explicit Request with no Limit picks up the namespace
			// default Limit, and the explicit Request is never touched.
			name:     "explicit request with no limit picks up Default for the limit only",
			requests: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("128Mi")},
			limits:   corev1.ResourceList{},
			limitRangeItems: []corev1.LimitRangeItem{
				containerItem(
					corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("512Mi")},
					corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("256Mi")},
				),
			},
			wantRequests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
			wantLimits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi"),
			},
			wantLimitsFromLimitRange: map[corev1.ResourceName]bool{corev1.ResourceMemory: true},
		},
		{
			name:     "a container with fully explicit values is never touched by LimitRange",
			requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("250m"), corev1.ResourceMemory: resource.MustParse("128Mi")},
			limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("500m"), corev1.ResourceMemory: resource.MustParse("256Mi")},
			limitRangeItems: []corev1.LimitRangeItem{
				containerItem(
					corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1"), corev1.ResourceMemory: resource.MustParse("1Gi")},
					corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m"), corev1.ResourceMemory: resource.MustParse("64Mi")},
				),
			},
			wantRequests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("250m"), corev1.ResourceMemory: resource.MustParse("128Mi")},
			wantLimits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("500m"), corev1.ResourceMemory: resource.MustParse("256Mi")},
		},
		{
			name:     "cpu and memory are resolved independently",
			requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m")}, // memory request unset
			limits:   corev1.ResourceList{},                                               // both limits unset
			limitRangeItems: []corev1.LimitRangeItem{
				containerItem(
					corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1"), corev1.ResourceMemory: resource.MustParse("512Mi")},
					corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("256Mi")}, // no cpu defaultRequest
				),
			},
			wantRequests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"), // untouched, explicit
				corev1.ResourceMemory: resource.MustParse("256Mi"),
			},
			wantLimits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1"),
				corev1.ResourceMemory: resource.MustParse("512Mi"),
			},
			wantRequestsFromLimitRange: map[corev1.ResourceName]bool{corev1.ResourceMemory: true},
			wantLimitsFromLimitRange:   map[corev1.ResourceName]bool{corev1.ResourceCPU: true, corev1.ResourceMemory: true},
		},
		{
			name:     "a Pod-typed LimitRangeItem is ignored, not applied to the container",
			requests: corev1.ResourceList{},
			limits:   corev1.ResourceList{},
			limitRangeItems: []corev1.LimitRangeItem{
				{
					Type:    corev1.LimitTypePod,
					Default: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("512Mi")},
				},
			},
			wantRequests: corev1.ResourceList{},
			wantLimits:   corev1.ResourceList{},
		},
		{
			name:     "multiple LimitRange items: first applicable item wins per resource, gaps only filled",
			requests: corev1.ResourceList{},
			limits:   corev1.ResourceList{},
			limitRangeItems: []corev1.LimitRangeItem{
				containerItem(corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("256Mi")}, nil),
				containerItem(corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("1Gi")}, nil), // should be ignored for memory, already filled
			},
			wantLimits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("256Mi"),
			},
			wantRequests:             corev1.ResourceList{},
			wantLimitsFromLimitRange: map[corev1.ResourceName]bool{corev1.ResourceMemory: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRequests, gotLimits, gotRequestsFromLimitRange, gotLimitsFromLimitRange := resolveEffectiveResources(tt.requests, tt.limits, tt.limitRangeItems)

			assert.True(t, quantitiesEqual(tt.wantRequests, gotRequests), "requests: want %v got %v", tt.wantRequests, gotRequests)
			assert.True(t, quantitiesEqual(tt.wantLimits, gotLimits), "limits: want %v got %v", tt.wantLimits, gotLimits)
			assert.Equal(t, tt.wantRequestsFromLimitRange, gotRequestsFromLimitRange)
			assert.Equal(t, tt.wantLimitsFromLimitRange, gotLimitsFromLimitRange)
		})
	}
}

// quantitiesEqual compares two ResourceLists by semantic quantity equality (via Cmp) rather than
// struct equality, since resource.Quantity carries a private cached string representation that
// can differ between two values constructed differently but representing the same amount.
func quantitiesEqual(a, b corev1.ResourceList) bool {
	if len(a) != len(b) {
		return false
	}
	for name, qa := range a {
		qb, ok := b[name]
		if !ok || qa.Cmp(qb) != 0 {
			return false
		}
	}
	return true
}

func Test_Summarizer_containerLimitRangeItems(t *testing.T) {
	t.Run("namespace with no LimitRange returns an empty, non-nil slice", func(t *testing.T) {
		kubeClient := kube.GetMockClient()
		s := &Summarizer{options: options{kubeClient: kubeClient}}

		got := s.containerLimitRangeItems("empty-namespace")
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})

	t.Run("only Container-typed items across possibly multiple LimitRange objects are returned", func(t *testing.T) {
		kubeClient := kube.GetMockClient()
		s := &Summarizer{options: options{kubeClient: kubeClient}}

		_, err := kubeClient.Client.CoreV1().LimitRanges("testing").Create(context.TODO(), &corev1.LimitRange{
			ObjectMeta: metav1.ObjectMeta{Name: "lr-a", Namespace: "testing"},
			Spec: corev1.LimitRangeSpec{
				Limits: []corev1.LimitRangeItem{
					{
						Type:    corev1.LimitTypePod,
						Default: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("1Gi")},
					},
					{
						Type:    corev1.LimitTypeContainer,
						Default: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("512Mi")},
					},
				},
			},
		}, metav1.CreateOptions{})
		assert.NoError(t, err)

		_, err = kubeClient.Client.CoreV1().LimitRanges("testing").Create(context.TODO(), &corev1.LimitRange{
			ObjectMeta: metav1.ObjectMeta{Name: "lr-b", Namespace: "testing"},
			Spec: corev1.LimitRangeSpec{
				Limits: []corev1.LimitRangeItem{
					{
						Type:           corev1.LimitTypeContainer,
						DefaultRequest: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m")},
					},
				},
			},
		}, metav1.CreateOptions{})
		assert.NoError(t, err)

		got := s.containerLimitRangeItems("testing")
		assert.Len(t, got, 2, "the Pod-typed item must be excluded")
		for _, item := range got {
			assert.Equal(t, corev1.LimitTypeContainer, item.Type)
		}

		// a different, unrelated namespace must not see these items
		other := s.containerLimitRangeItems("other-namespace")
		assert.Empty(t, other)
	})

	t.Run("result is cached so a second call for the same namespace doesn't hit the API again", func(t *testing.T) {
		kubeClient := kube.GetMockClient()
		s := &Summarizer{options: options{kubeClient: kubeClient}}

		_, err := kubeClient.Client.CoreV1().LimitRanges("testing").Create(context.TODO(), &corev1.LimitRange{
			ObjectMeta: metav1.ObjectMeta{Name: "lr-a", Namespace: "testing"},
			Spec: corev1.LimitRangeSpec{
				Limits: []corev1.LimitRangeItem{
					{Type: corev1.LimitTypeContainer, Default: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("512Mi")}},
				},
			},
		}, metav1.CreateOptions{})
		assert.NoError(t, err)

		first := s.containerLimitRangeItems("testing")
		assert.Len(t, first, 1)

		// delete the LimitRange from the fake API -- if containerLimitRangeItems were not
		// caching, the second call would now see zero items
		assert.NoError(t, kubeClient.Client.CoreV1().LimitRanges("testing").Delete(context.TODO(), "lr-a", metav1.DeleteOptions{}))

		second := s.containerLimitRangeItems("testing")
		assert.Len(t, second, 1, "expected cached result to be reused instead of re-querying the API")
	})
}
