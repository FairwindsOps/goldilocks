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

package helpers

import (
	uuid "github.com/satori/go.uuid"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func PrintResource(quant resource.Quantity) string {
	if quant.IsZero() {
		return "Not Set"
	}
	return quant.String()
}

func GetStatus(existing resource.Quantity, recommendation resource.Quantity, style string) string {
	if existing.IsZero() {
		switch style {
		case "text":
			return "error - not set"
		case "icon":
			return "fa-exclamation error"
		default:
			return ""
		}
	}

	comparison := existing.Cmp(recommendation)
	if comparison == 0 {
		switch style {
		case "text":
			return "equal"
		case "icon":
			return "fa-equals success"
		default:
			return ""
		}
	}
	if comparison < 0 {
		switch style {
		case "text":
			return "less than"
		case "icon":
			return "fa-less-than warning"
		default:
			return ""
		}
	}
	if comparison > 0 {
		switch style {
		case "text":
			return "greater than"
		case "icon":
			return "fa-greater-than warning"
		default:
			return ""
		}
	}
	return ""
}

func GetStatusRange(existing, lower, upper resource.Quantity, style string) string {
	if existing.IsZero() {
		switch style {
		case "text":
			return "error - not set"
		case "icon":
			return "fa-exclamation error"
		default:
			return ""
		}
	}

	comparisonLower := existing.Cmp(lower)
	comparisonUpper := existing.Cmp(upper)
	if comparisonUpper <= 0 && comparisonLower >= 0 {
		switch style {
		case "text":
			return "equal"
		case "icon":
			return "fa-equals success"
		default:
			return ""
		}
	}

	if comparisonLower < 0 {
		switch style {
		case "text":
			return "less than"
		case "icon":
			return "fa-less-than warning"
		}
	}

	if comparisonUpper > 0 {
		switch style {
		case "text":
			return "greater than"
		case "icon":
			return "fa-greater-than warning"
		}
	}

	return ""
}

func ResourceName(name string) corev1.ResourceName {
	return corev1.ResourceName(name)
}

func GetUUID() string {
	return uuid.NewV4().String()
}
