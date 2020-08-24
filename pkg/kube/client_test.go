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

package kube

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetNamespace(t *testing.T) {
	kubeClient := GetMockClient()

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}

	_, errNotFound := GetNamespace(kubeClient, "nothere")
	assert.EqualError(t, errNotFound, "namespaces \"nothere\" not found")

	_, err := kubeClient.Client.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
	assert.NoError(t, err)

	got, err := GetNamespace(kubeClient, "test")
	assert.NoError(t, err)
	assert.EqualValues(t, got, namespace)
}
