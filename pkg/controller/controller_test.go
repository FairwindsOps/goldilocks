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

package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_objectMeta(t *testing.T) {
	tests := []struct {
		name string
		obj  interface{}
		want metav1.ObjectMeta
	}{
		{
			name: "Namespace with Labels",
			obj: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ns",
					Namespace: "test",
					Labels: map[string]string{
						"goldilocks.fairwinds.com/enabled": "True",
					},
				},
			},
			want: metav1.ObjectMeta{
				Name: "ns",
				Labels: map[string]string{
					"goldilocks.fairwinds.com/enabled": "True",
				},
				Namespace: "test",
			},
		},
		{
			name: "Deployment",
			obj: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "deployment",
					Namespace: "test",
				},
			},
			want: metav1.ObjectMeta{
				Namespace: "test",
				Name:      "deployment",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.EqualValues(t, objectMeta(tt.obj), tt.want)
		})
	}
}
