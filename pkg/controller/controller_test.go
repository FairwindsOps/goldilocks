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
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/fairwindsops/goldilocks/pkg/metrics"
)

func Test_objectMeta(t *testing.T) {
	tests := []struct {
		name string
		obj  any
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
			name: "Pod",
			obj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod",
					Namespace: "test",
				},
			},
			want: metav1.ObjectMeta{
				Namespace: "test",
				Name:      "pod",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.EqualValues(t, objectMeta(tt.obj), tt.want)
		})
	}
}

func Test_createController_IncrementsEventMetrics(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return fakeClient.CoreV1().Pods("").List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return fakeClient.CoreV1().Pods("").Watch(context.TODO(), options)
			},
		},
		&corev1.Pod{},
		0,
		cache.Indexers{},
	)

	watcher := createController(fakeClient, informer, "pod")

	before := testutil.ToFloat64(metrics.EventsProcessedTotal.WithLabelValues("pod", "create"))

	stop := make(chan struct{})
	defer close(stop)
	go informer.Run(stop)
	assert.True(t, cache.WaitForCacheSync(stop, informer.HasSynced))

	_, err := fakeClient.CoreV1().Pods("default").Create(context.TODO(), &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "default"},
	}, metav1.CreateOptions{})
	assert.NoError(t, err)

	assert.Eventually(t, func() bool {
		return testutil.ToFloat64(metrics.EventsProcessedTotal.WithLabelValues("pod", "create")) >= before+1
	}, 2*time.Second, 10*time.Millisecond, "expected the pod create event to increment the events_processed_total metric")

	watcher.wq.ShutDown()
}
