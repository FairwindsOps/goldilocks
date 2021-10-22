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
	LabelBase = "goldilocks.fairwinds.com"
	// VpaEnabledLabel is the label used to indicate that Goldilocks is enabled.
	VpaEnabledLabel = LabelBase + "/" + "enabled"
	// VpaUpdateModeKey is the label used to indicate the vpa update mode.
	VpaUpdateModeKey = LabelBase + "/" + "vpa-update-mode"
	// WorkloadExcludeContainersAnnotation is the label used to exclude container names from being reported.
	WorkloadExcludeContainersAnnotation = LabelBase + "/" + "exclude-containers"
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

// FormatResourceList scales the units of a ResourceList so that they are
// human readable
func FormatResourceList(rl v1.ResourceList) v1.ResourceList {
	memoryScales := []resource.Scale{
		resource.Kilo,
		resource.Mega,
		resource.Giga,
		resource.Tera,
	}
	if mem, exists := rl[v1.ResourceMemory]; exists {
		i := 0
		maxAllowableStringLen := 5
		for len(mem.String()) > maxAllowableStringLen && i < len(memoryScales)-1 {
			mem.RoundUp(memoryScales[i])
			i++
		}
		rl[v1.ResourceMemory] = mem
	}
	return rl
}
