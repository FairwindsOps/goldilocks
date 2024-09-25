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
	"fmt"
	"time"

	"k8s.io/klog/v2"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	rt "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/fairwindsops/goldilocks/pkg/handler"
	"github.com/fairwindsops/goldilocks/pkg/kube"
	"github.com/fairwindsops/goldilocks/pkg/utils"
)

// KubeResourceWatcher contains the informer that watches Kubernetes objects and the queue that processes updates.
type KubeResourceWatcher struct {
	kubeClient kubernetes.Interface
	informer   cache.SharedIndexInformer
	wq         workqueue.TypedRateLimitingInterface[any]
}

// Watch tells the KubeResourceWatcher to start waiting for events
func (watcher *KubeResourceWatcher) Watch(term <-chan struct{}) {
	klog.Infof("Starting watcher.")

	defer watcher.wq.ShutDown()
	defer rt.HandleCrash()

	go watcher.informer.Run(term)

	if !cache.WaitForCacheSync(term, watcher.HasSynced) {
		rt.HandleError(fmt.Errorf("timeout waiting for cache sync"))
		return
	}

	klog.Infof("Watcher synced.")
	wait.Until(watcher.waitForEvents, time.Second, term)
}

func (watcher *KubeResourceWatcher) waitForEvents() {
	// just keep running forever
	for watcher.next() {
	}
}

// HasSynced determines whether the informer has synced
func (watcher *KubeResourceWatcher) HasSynced() bool {
	return watcher.informer.HasSynced()
}

// LastSyncResourceVersion returns the last sync resource version
func (watcher *KubeResourceWatcher) LastSyncResourceVersion() string {
	return watcher.informer.LastSyncResourceVersion()
}

func (watcher *KubeResourceWatcher) process(evt utils.Event) error {
	info, _, err := watcher.informer.GetIndexer().GetByKey(evt.Key)

	if err != nil {
		//TODO - need some better error handling here
		return err
	}

	handler.OnUpdate(info, evt)
	return nil
}

func (watcher *KubeResourceWatcher) next() bool {
	evt, err := watcher.wq.Get()

	if err {
		return false
	}

	defer watcher.wq.Done(evt)
	processErr := watcher.process(evt.(utils.Event))
	if processErr != nil {
		// limit the number of retries
		if watcher.wq.NumRequeues(evt) < 5 {
			klog.Errorf("Error running queued item %s: %v", evt.(utils.Event).Key, processErr)
			klog.Infof("Retry processing item %s", evt.(utils.Event).Key)
			watcher.wq.AddRateLimited(evt)
		} else {
			klog.Errorf("Giving up trying to run queued item %s: %v", evt.(utils.Event).Key, processErr)
			watcher.wq.Forget(evt)
			rt.HandleError(processErr)
		}
	}
	return true
}

// NewController starts a controller for watching Kubernetes objects.
func NewController(stop <-chan bool) {
	klog.Info("Starting controller.")
	kubeClient := kube.GetInstance()

	klog.Infof("Creating watcher for Pods.")
	PodInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return kubeClient.Client.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return kubeClient.Client.CoreV1().Pods("").Watch(context.TODO(), metav1.ListOptions{})
			},
		},
		&corev1.Pod{},
		0,
		cache.Indexers{},
	)

	PodWatcher := createController(kubeClient.Client, PodInformer, "pod")
	pTerm := make(chan struct{})
	defer close(pTerm)
	go PodWatcher.Watch(pTerm)

	klog.Infof("Creating watcher for Namespaces.")
	NSInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return kubeClient.Client.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return kubeClient.Client.CoreV1().Namespaces().Watch(context.TODO(), metav1.ListOptions{})
			},
		},
		&corev1.Namespace{},
		0,
		cache.Indexers{},
	)

	NSWatcher := createController(kubeClient.Client, NSInformer, "namespace")
	nsTerm := make(chan struct{})
	defer close(nsTerm)
	go NSWatcher.Watch(nsTerm)

	if <-stop {
		klog.Info("Shutting down controller.")
		return
	}

}

func createController(kubeClient kubernetes.Interface, informer cache.SharedIndexInformer, resource string) *KubeResourceWatcher {
	klog.Infof("Creating controller for resource type %s", resource)
	wq := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[any](), resource)

	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			var evt utils.Event
			var err error
			evt.Key, err = cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				klog.Errorf("Error handling add event")
				return
			}
			evt.EventType = "create"
			evt.ResourceType = resource
			evt.Namespace = objectMeta(obj).Namespace
			klog.V(2).Infof("%s/%s has been added.", resource, evt.Key)
			wq.Add(evt)
		},
		DeleteFunc: func(obj interface{}) {
			var evt utils.Event
			var err error
			evt.Key, err = cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				klog.Errorf("Error handling delete event")
				return
			}
			evt.EventType = "delete"
			evt.ResourceType = resource
			evt.Namespace = objectMeta(obj).Namespace
			klog.V(2).Infof("%s/%s has been deleted.", resource, evt.Key)
			wq.Add(evt)
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			var evt utils.Event
			var err error
			evt.Key, err = cache.MetaNamespaceKeyFunc(new)
			if err != nil {
				klog.Errorf("Error handling update event")
				return
			}
			evt.EventType = "update"
			evt.ResourceType = resource
			evt.Namespace = objectMeta(new).Namespace
			klog.V(8).Infof("%s/%s has been updated.", resource, evt.Key)
			wq.Add(evt)
		},
	})

	if err != nil {
		panic(err)
	}

	return &KubeResourceWatcher{
		kubeClient: kubeClient,
		informer:   informer,
		wq:         wq,
	}
}

func objectMeta(obj interface{}) metav1.ObjectMeta {
	var meta metav1.ObjectMeta

	switch object := obj.(type) {
	case *corev1.Namespace:
		meta = object.ObjectMeta
	case *corev1.Pod:
		meta = object.ObjectMeta
	}
	return meta
}
