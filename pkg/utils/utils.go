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

package utils

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	// LabelBase is the string that will be used for labels on namespaces
	LabelOrAnnotationBase = "goldilocks.fairwinds.com"
	// VpaEnabledLabel is the label used to indicate that Goldilocks is enabled.
	VpaEnabledLabel = LabelOrAnnotationBase + "/" + "enabled"
	// VpaUpdateModeKey is the label used to indicate the vpa update mode.
	VpaUpdateModeKey = LabelOrAnnotationBase + "/" + "vpa-update-mode"
	// VpaMinReplicas is the annotation to use to define minimum replicas for eviction of a VPA
	VpaMinReplicasAnnotation = LabelOrAnnotationBase + "/" + "vpa-min-replicas"
	// DeploymentExcludeContainersAnnotation is the label used to exclude container names from being reported.
	WorkloadExcludeContainersAnnotation = LabelOrAnnotationBase + "/" + "exclude-containers"
	// VpaResourcePolicyAnnotation is the annotation use to define the json configuration of PodResourcePolicy section of a vpa
	VpaResourcePolicyAnnotation = LabelOrAnnotationBase + "/" + "vpa-resource-policy"
)

// VPALabels is a set of default labels that get placed on every VPA.
var VPALabels = map[string]string{
	"creator": "Fairwinds",
	"source":  "goldilocks",
}

// An Event represents an update of a Kubernetes object and contains metadata about the update.
type Event struct {
	Key          string // A key identifying the object.  This is in the format <object-type>/<object-name>
	EventType    string // The type of event - update, delete, or create
	Namespace    string // The namespace of the event's object
	ResourceType string // The type of resource that was updated.
}

// UniqueString returns a unique string from a slice.
func UniqueString(stringSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range stringSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

// Difference returns the difference betwee two string slices.
func Difference(a, b []string) (diff []string) {
	m := make(map[string]bool)

	for _, item := range b {
		m[item] = true
	}

	for _, item := range a {
		if _, ok := m[item]; !ok {
			diff = append(diff, item)
		}
	}
	return
}

// FormatResourceList ensures that memory resources are displayed in BinarySI
// (Ki, Mi, Gi) format to avoid confusion with Decimal units (k, M, G).
// It rounds values to the nearest unit for readability (Ki for small values, Mi for larger ones).
func FormatResourceList(rl v1.ResourceList) v1.ResourceList {
	if mem, exists := rl[v1.ResourceMemory]; exists {
		mem.Format = resource.BinarySI
		val := mem.Value()

		const Ki = int64(1024)
		const Mi = int64(1024 * 1024)

		// Determine the rounding unit (step)
		var step int64
		if val < Mi {
			// For small values (< 1 MiB), round to nearest Ki
			step = Ki
		} else {
			// For standard values (>= 1 MiB), round to nearest Mi
			step = Mi
		}

		if step > 0 && val > 0 {
			// Rounding formula: (val + step - 1) / step * step
			rounded := ((val + step - 1) / step) * step
			mem.Set(rounded)
		}

        // Force the internal string cache to be regenerated.
		// When mem.Set() is called, the internal string representation (private field 's') is cleared.
		// Calling .String() forces it to be recomputed and stored.
		// This is required for DeepEqual comparisons in unit tests to pass, as they compare private fields.
        _ = mem.String()

		rl[v1.ResourceMemory] = mem
	}
	return rl
}
