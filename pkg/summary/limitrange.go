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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

// containerLimitRangeItems returns every Container-typed LimitRangeItem found across all
// LimitRange objects in the given namespace, in the order returned by the API. The result is
// cached on the Summarizer for the remainder of its lifetime so that a namespace containing
// many VPAs/workloads/containers only ever triggers a single LimitRanges List call, instead of
// one per container.
//
// A namespace with no LimitRange objects (by far the common case) returns an empty, non-nil
// slice and callers treat that as a no-op.
func (s *Summarizer) containerLimitRangeItems(namespace string) []corev1.LimitRangeItem {
	if s.limitRangeItemsByNamespace == nil {
		s.limitRangeItemsByNamespace = map[string][]corev1.LimitRangeItem{}
	}
	if items, ok := s.limitRangeItemsByNamespace[namespace]; ok {
		return items
	}

	items := []corev1.LimitRangeItem{}
	limitRanges, err := s.kubeClient.Client.CoreV1().LimitRanges(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		// Don't fail the whole summary just because LimitRanges couldn't be listed (e.g. an
		// RBAC restriction) -- fall back to behaving as if there were none, which is the same
		// "Not Set" behavior this feature is layered on top of.
		klog.Errorf("unable to list LimitRanges in namespace '%s', proceeding as if none exist: %v", namespace, err)
	} else {
		for _, lr := range limitRanges.Items {
			for _, item := range lr.Spec.Limits {
				// Only "Container" typed items apply to a specific container's requests/limits.
				// "Pod" and "PersistentVolumeClaim" typed items constrain other things and are
				// not relevant to what this dashboard displays.
				if item.Type == corev1.LimitTypeContainer {
					items = append(items, item)
				}
			}
		}
	}

	s.limitRangeItemsByNamespace[namespace] = items
	return items
}

// resolveEffectiveResources fills in the *effective* requests/limits a container would actually
// receive from Kubernetes, given its own explicit resources.Requests/resources.Limits plus the
// Container-typed LimitRangeItems that apply in its namespace. Values that are already explicit
// on the container are always left untouched; a LimitRange only ever fills in a gap.
//
// The two maps returned indicate, per resource name (e.g. "cpu", "memory"), whether the
// corresponding entry in the returned effRequests/effLimits was *not* present on the container
// itself and was instead resolved from a namespace LimitRange.
//
// This intentionally reproduces two distinct steps of real Kubernetes behavior, in the same
// order the API server performs them, because the order matters for a case that's easy to get
// backwards:
//
//  1. Pod-level API defaulting (`SetDefaults_Pod` in k8s.io/kubernetes/pkg/apis/core/v1,
//     https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/core/v1/defaults.go): if a
//     container explicitly sets a Limit for a resource but leaves the Request unset, the Request
//     is defaulted to equal that explicit Limit. This is pure type defaulting that runs at
//     decode time, before any admission plugin (including LimitRanger) ever sees the object, and
//     it only ever looks at what the container itself already specifies -- never at a value a
//     LimitRange might go on to supply.
//  2. LimitRanger admission (plugin/pkg/admission/limitranger): for any resource whose Limit is
//     *still* unset after step 1, apply the namespace LimitRange's `Default`. For any resource
//     whose Request is *still* unset after step 1, apply the namespace LimitRange's
//     `DefaultRequest`. Because step 1 has already run, a Limit that only becomes set here (via
//     `Default`) is never copied into a still-unset Request -- `DefaultRequest` is the only thing
//     that can fill an unset Request at this stage. This is the trap: it's tempting to assume
//     "Limit ends up set -> Request defaults to it" applies regardless of *how* the Limit was
//     set, but Kubernetes' own ordering means that only holds for an explicit Limit.
//
// Multiple LimitRange objects (or multiple Container-typed items) in one namespace are also
// possible. Per the Kubernetes docs (https://kubernetes.io/docs/concepts/policy/limit-range/,
// "If two or more LimitRange objects exist in the namespace, it is not deterministic which
// default value will be applied."), real admission behavior is explicitly non-deterministic
// here. For display purposes we approximate LimitRanger's actual "only fill what's still
// missing" mechanics (plugin/pkg/admission/limitranger/admission.go's mergeContainerResources)
// applied across items in list order: the first applicable item found supplies a given resource,
// later ones are ignored for that resource. limitRangeItems is expected to already be filtered
// to Container-typed items (see containerLimitRangeItems), in the order they should be
// considered.
func resolveEffectiveResources(requests, limits corev1.ResourceList, limitRangeItems []corev1.LimitRangeItem) (effRequests, effLimits corev1.ResourceList, requestsFromLimitRange, limitsFromLimitRange map[corev1.ResourceName]bool) {
	effRequests = requests.DeepCopy()
	effLimits = limits.DeepCopy()
	if effRequests == nil {
		effRequests = corev1.ResourceList{}
	}
	if effLimits == nil {
		effLimits = corev1.ResourceList{}
	}

	// Step 1: mimic SetDefaults_Pod -- an explicit Limit with no explicit Request becomes the
	// Request. Deliberately iterates over the container's own original `limits`, not
	// `effLimits`, so that a Limit added below by a LimitRange Default is never used as a
	// source for this copy.
	for name, explicitLimit := range limits {
		if _, hasRequest := effRequests[name]; !hasRequest {
			effRequests[name] = explicitLimit.DeepCopy()
		}
	}

	// Step 2: apply namespace LimitRange defaults for anything still unset. First applicable
	// item wins per resource name, independently for Default (limits) and DefaultRequest
	// (requests).
	for _, item := range limitRangeItems {
		// Defense in depth: only "Container" typed items constrain a single container's
		// requests/limits. Callers (containerLimitRangeItems) are expected to pre-filter to
		// these, but a stray "Pod"/"PersistentVolumeClaim" typed item must never be applied
		// here either.
		if item.Type != corev1.LimitTypeContainer {
			continue
		}
		for name, def := range item.Default {
			if _, has := effLimits[name]; !has {
				effLimits[name] = def.DeepCopy()
				if limitsFromLimitRange == nil {
					limitsFromLimitRange = map[corev1.ResourceName]bool{}
				}
				limitsFromLimitRange[name] = true
			}
		}
		for name, def := range item.DefaultRequest {
			if _, has := effRequests[name]; !has {
				effRequests[name] = def.DeepCopy()
				if requestsFromLimitRange == nil {
					requestsFromLimitRange = map[corev1.ResourceName]bool{}
				}
				requestsFromLimitRange[name] = true
			}
		}
	}

	return effRequests, effLimits, requestsFromLimitRange, limitsFromLimitRange
}
