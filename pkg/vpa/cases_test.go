// Copyright 2019 Fairwinds
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

package vpa

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
