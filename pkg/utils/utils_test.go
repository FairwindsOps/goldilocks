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
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestUniqueString(t *testing.T) {
	for _, tc := range testUniqueStringCases {
		res := UniqueString(tc.testData)
		assert.Equal(t, res, tc.expected)
	}
}

var testUniqueStringCases = []struct {
	description string
	testData    []string
	expected    []string
}{
	{
		description: "three duplicates, one output",
		testData:    []string{"test", "test", "test"},
		expected:    []string{"test"},
	},
	{
		description: "no duplicates",
		testData:    []string{"one", "two", "three"},
		expected:    []string{"one", "two", "three"},
	},
}

func TestDifference(t *testing.T) {
	for _, tc := range testDifferenceCases {
		res := Difference(tc.testData1, tc.testData2)
		assert.Equal(t, res, tc.expected)
	}
}

var empty []string

var testDifferenceCases = []struct {
	description string
	testData1   []string
	testData2   []string
	expected    []string
}{
	{
		description: "empty case",
		testData1:   []string{"a", "b", "c"},
		testData2:   []string{"a", "b", "c"},
		expected:    empty,
	},
	{
		description: "extra item on right",
		testData1:   []string{"a", "b"},
		testData2:   []string{"a", "b", "c"},
		expected:    empty,
	},
	{
		description: "extra item on left",
		testData1:   []string{"a", "b", "c"},
		testData2:   []string{"a", "b"},
		expected:    []string{"c"},
	},
}

func TestFormatResourceList(t *testing.T) {
	for _, tc := range testFormatResourceCases {
		res := FormatResourceList(tc.testData, tc.useMemoryBinarySI)
		resource := res[tc.resourceType]
		got := resource.String()
		assert.Equal(t, tc.expected, got)
	}
}

var testFormatResourceCases = []struct {
	description       string
	testData          v1.ResourceList
	resourceType      v1.ResourceName
	useMemoryBinarySI bool
	expected          string
}{
	{
		description: "Unmodified cpu",
		testData: v1.ResourceList{
			"cpu": resource.MustParse("1"),
		},
		resourceType: "cpu",
		expected:     "1",
	},
	{
		description: "Unmodified memory",
		testData: v1.ResourceList{
			"memory": resource.MustParse("1Mi"),
		},
		resourceType: "memory",
		expected:     "1Mi",
	},
	{
		description: "Memory in too large of units",
		testData: v1.ResourceList{
			"memory": resource.MustParse("123456k"),
		},
		resourceType:      "memory",
		useMemoryBinarySI: false,
		expected:          "124M",
	},
	{
		description: "Memory in too large of units",
		testData: v1.ResourceList{
			"memory": resource.MustParse("123456k"),
		},
		resourceType:      "memory",
		useMemoryBinarySI: true,
		expected:          "119Mi",
	},
}
