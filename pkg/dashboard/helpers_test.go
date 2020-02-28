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

package dashboard

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
)

func Test_getUUID(t *testing.T) {
	var validUUID = regexp.MustCompile("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$")
	match := validUUID.MatchString(getUUID())
	assert.Equal(t, match, true)
}

func Test_printResource(t *testing.T) {
	tests := []struct {
		name     string
		quantity resource.Quantity
		want     string
	}{
		{
			name:     "Blank",
			quantity: resource.Quantity{},
			want:     "Not Set",
		},
		{
			name:     "cpu",
			quantity: *resource.NewMilliQuantity(25, resource.DecimalSI),
			want:     "25m",
		},
		{
			name:     "mem",
			quantity: *resource.NewQuantity(5*1024*1024*1024, resource.BinarySI),
			want:     "5Gi",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := printResource(tt.quantity)
			assert.Equal(t, got, tt.want)
		})
	}
}

func Test_getStatus(t *testing.T) {
	type args struct {
		existing       resource.Quantity
		recommendation resource.Quantity
		style          string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "icon gt",
			args: args{
				existing:       *resource.NewMilliQuantity(50, resource.DecimalSI),
				recommendation: *resource.NewMilliQuantity(25, resource.DecimalSI),
				style:          "icon",
			},
			want: "fa-greater-than warning",
		},
		{
			name: "text gt",
			args: args{
				existing:       *resource.NewMilliQuantity(50, resource.DecimalSI),
				recommendation: *resource.NewMilliQuantity(25, resource.DecimalSI),
				style:          "text",
			},
			want: "greater than",
		},
		{
			name: "icon lt",
			args: args{
				existing:       *resource.NewMilliQuantity(25, resource.DecimalSI),
				recommendation: *resource.NewMilliQuantity(50, resource.DecimalSI),
				style:          "icon",
			},
			want: "fa-less-than warning",
		},
		{
			name: "text lt",
			args: args{
				existing:       *resource.NewMilliQuantity(25, resource.DecimalSI),
				recommendation: *resource.NewMilliQuantity(50, resource.DecimalSI),
				style:          "text",
			},
			want: "less than",
		},
		{
			name: "text equal",
			args: args{
				existing:       *resource.NewMilliQuantity(25, resource.DecimalSI),
				recommendation: *resource.NewMilliQuantity(25, resource.DecimalSI),
				style:          "text",
			},
			want: "equal",
		},
		{
			name: "icon equal",
			args: args{
				existing:       *resource.NewMilliQuantity(25, resource.DecimalSI),
				recommendation: *resource.NewMilliQuantity(25, resource.DecimalSI),
				style:          "icon",
			},
			want: "fa-equals success",
		},
		{
			name: "icon not set",
			args: args{
				existing:       resource.Quantity{},
				recommendation: resource.Quantity{},
				style:          "icon",
			},
			want: "fa-exclamation error",
		},
		{
			name: "text not set",
			args: args{
				existing:       resource.Quantity{},
				recommendation: resource.Quantity{},
				style:          "text",
			},
			want: "error - not set",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getStatus(tt.args.existing, tt.args.recommendation, tt.args.style)
			assert.Equal(t, got, tt.want)
		})
	}
}

func Test_getStatusRange(t *testing.T) {
	type args struct {
		existing resource.Quantity
		lower    resource.Quantity
		upper    resource.Quantity
		style    string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "icon gt",
			args: args{
				existing: *resource.NewMilliQuantity(50, resource.DecimalSI),
				lower:    *resource.NewMilliQuantity(25, resource.DecimalSI),
				upper:    *resource.NewMilliQuantity(30, resource.DecimalSI),
				style:    "icon",
			},
			want: "fa-greater-than warning",
		},
		{
			name: "text gt",
			args: args{
				existing: *resource.NewMilliQuantity(50, resource.DecimalSI),
				lower:    *resource.NewMilliQuantity(25, resource.DecimalSI),
				upper:    *resource.NewMilliQuantity(30, resource.DecimalSI),
				style:    "text",
			},
			want: "greater than",
		},
		{
			name: "icon lt",
			args: args{
				existing: *resource.NewMilliQuantity(25, resource.DecimalSI),
				lower:    *resource.NewMilliQuantity(50, resource.DecimalSI),
				upper:    *resource.NewMilliQuantity(75, resource.DecimalSI),
				style:    "icon",
			},
			want: "fa-less-than warning",
		},
		{
			name: "text lt",
			args: args{
				existing: *resource.NewMilliQuantity(25, resource.DecimalSI),
				lower:    *resource.NewMilliQuantity(50, resource.DecimalSI),
				upper:    *resource.NewMilliQuantity(75, resource.DecimalSI),
				style:    "text",
			},
			want: "less than",
		},
		{
			name: "text equal",
			args: args{
				existing: *resource.NewMilliQuantity(50, resource.DecimalSI),
				lower:    *resource.NewMilliQuantity(25, resource.DecimalSI),
				upper:    *resource.NewMilliQuantity(75, resource.DecimalSI),
				style:    "text",
			},
			want: "equal",
		},
		{
			name: "icon equal",
			args: args{
				existing: *resource.NewMilliQuantity(50, resource.DecimalSI),
				lower:    *resource.NewMilliQuantity(25, resource.DecimalSI),
				upper:    *resource.NewMilliQuantity(75, resource.DecimalSI),
				style:    "icon",
			},
			want: "fa-equals success",
		},
		{
			name: "icon not set",
			args: args{
				existing: resource.Quantity{},
				upper:    resource.Quantity{},
				lower:    resource.Quantity{},
				style:    "icon",
			},
			want: "fa-exclamation error",
		},
		{
			name: "text not set",
			args: args{
				existing: resource.Quantity{},
				lower:    resource.Quantity{},
				upper:    resource.Quantity{},
				style:    "text",
			},
			want: "error - not set",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getStatusRange(tt.args.existing, tt.args.lower, tt.args.upper, tt.args.style)
			assert.Equal(t, got, tt.want)
		})
	}
}
