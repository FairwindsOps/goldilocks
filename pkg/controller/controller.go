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
	"github.com/fairwindsops/goldilocks/pkg/vpa"
)

// KubeResourceWatcher contains the informer that watches Kubernetes objects and the queue that processes updates.
type KubeResourceWatcher struct {
	kubeClient         kubernetes.Interface
	informer           cache.SharedIndexInformer
	wq                 workqueue.TypedRateLimitingInterface[any]
	lastWatchFailure   time.Time
	watchFailureCount  int
	controlPlanePause  time.Duration
	eventProcessingErrors int    // Track consecutive event processing errors
	lastEventError     time.Time // Track when the last event error occurred
	lastInformerRestart time.Time // Track informer restarts
	informerRestarts   int       // Count informer restarts
}

// Watch tells the KubeResourceWatcher to start waiting for events
func (watcher *KubeResourceWatcher) Watch(term <-chan struct{}) {
	// Determine current log level
	logLevel := 0
	for i := 10; i >= 0; i-- {
		if klog.V(klog.Level(i)).Enabled() {
			logLevel = i
			break
		}
	}
	klog.Infof("Starting goldilocks controller (log level V=%d)", logLevel)

	defer watcher.wq.ShutDown()
	defer rt.HandleCrash()

	// Record this informer restart
	watcher.recordInformerRestart()

	// Check if we should pause due to recent control plane pressure OR frequent informer restarts
	if watcher.shouldPauseForControlPlane() {
		klog.Warningf("CONTROL PLANE BACKOFF: Pausing watcher startup for %v due to control plane pressure (failure #%d, informer restarts #%d)", 
			watcher.controlPlanePause, watcher.watchFailureCount, watcher.informerRestarts)
		time.Sleep(watcher.controlPlanePause)
		klog.Warningf("CONTROL PLANE BACKOFF: Resuming watcher startup after %v pause", watcher.controlPlanePause)
	}

	go watcher.informer.Run(term)

	if !cache.WaitForCacheSync(term, watcher.HasSynced) {
		klog.Errorf("Cache sync timeout - this may indicate control plane pressure")
		// Record this as a potential control plane pressure indicator
		watcher.recordWatchFailure()
		rt.HandleError(fmt.Errorf("timeout waiting for cache sync"))
		return
	}

	klog.Infof("Watcher synced.")
	
	// Note: Watch stream errors from reflector.go are logged by client-go but don't trigger our backoff
	// because they're handled internally by the reflector with its own retry logic
	klog.V(1).Infof("Watch streams active - reflector errors visible in logs don't trigger backoff unless they cause downstream issues")
	
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
	// Check if we should pause due to recent control plane pressure
	if watcher.shouldPauseForControlPlane() {
		klog.Warningf("CONTROL PLANE BACKOFF: Pausing event processing for %v due to control plane pressure (failure #%d)", 
			watcher.controlPlanePause, watcher.watchFailureCount)
		time.Sleep(watcher.controlPlanePause)
		klog.Warningf("CONTROL PLANE BACKOFF: Resuming event processing after %v pause", watcher.controlPlanePause)
		// Reset the pause after using it to avoid repeated pauses for the same failure
		watcher.controlPlanePause = 0
	}

	info, _, err := watcher.informer.GetIndexer().GetByKey(evt.Key)

	if err != nil {
		// Enhanced error handling for better debugging
		klog.Errorf("Error getting object by key %s: %v", evt.Key, err)
		
		// Check if this is a control plane pressure related error (but not RBAC)
		if utils.IsRetryableError(err) && !utils.IsRBACError(err) {
			watcher.recordWatchFailure()
		}
		
		return err
	}

	// For namespace events that might trigger reconciliation, wait for any active reconciliation to complete
	if evt.ResourceType == "namespace" && (evt.EventType == "create" || evt.EventType == "update") {
		reconciler := vpa.GetInstance()
		if reconciler.IsReconciliationInProgress() {
			klog.V(3).Infof("Waiting for active reconciliation to complete before processing %s event for %s", evt.EventType, evt.Key)
			reconciler.WaitForReconciliationToComplete()
			klog.V(3).Infof("Active reconciliation completed, now processing %s event for %s", evt.EventType, evt.Key)
		}
	}

	// Process the event with improved error handling
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
		// Handling etcd pressure
		numRequeues := watcher.wq.NumRequeues(evt)
		maxRetries := 5

		if numRequeues < maxRetries {
			klog.Errorf("Error running queued item %s (attempt %d/%d): %v", evt.(utils.Event).Key, numRequeues+1, maxRetries, processErr)
			if utils.IsRetryableError(processErr) {
				klog.Infof("Detected retryable error for item %s, will retry with backoff", evt.(utils.Event).Key)
				// Record this as a potential control plane pressure indicator (but not RBAC errors)
				if !utils.IsRBACError(processErr) {
					watcher.recordWatchFailure()
					watcher.recordEventProcessingError()
				}
			} else {
				klog.Infof("Retrying processing item %s", evt.(utils.Event).Key)
				watcher.recordEventProcessingError()
			}
			watcher.wq.AddRateLimited(evt)
		} else {
			klog.Errorf("Giving up trying to run queued item %s after %d attempts: %v", evt.(utils.Event).Key, maxRetries, processErr)
			watcher.wq.Forget(evt)
			watcher.recordEventProcessingError()
			rt.HandleError(processErr)
		}
	} else {
		// Success - reset error counters and forget the item to reset rate limiting
		watcher.resetEventProcessingErrors()
		watcher.wq.Forget(evt)
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
				ctx, cancel := utils.CreateContextWithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				return kubeClient.Client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				ctx, cancel := utils.CreateContextWithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				return kubeClient.Client.CoreV1().Pods("").Watch(ctx, metav1.ListOptions{})
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
				ctx, cancel := utils.CreateContextWithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				return kubeClient.Client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				ctx, cancel := utils.CreateContextWithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				return kubeClient.Client.CoreV1().Namespaces().Watch(ctx, metav1.ListOptions{})
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
				klog.Errorf("Error handling add event: failed to get key for object %T: %v", obj, err)
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
			
			// Handle DeletedFinalStateUnknown objects
			if deletedObj, ok := obj.(cache.DeletedFinalStateUnknown); ok {
				klog.V(4).Infof("Handling DeletedFinalStateUnknown object, extracting original object")
				obj = deletedObj.Obj
			}
			
			evt.Key, err = cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				klog.V(4).Infof("Cannot get key for deleted object %T, skipping: %v", obj, err)
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
				klog.Errorf("Error handling update event: failed to get key for object %T: %v", new, err)
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

// shouldPauseForControlPlane determines if we should pause due to recent watch failures OR frequent informer restarts
func (watcher *KubeResourceWatcher) shouldPauseForControlPlane() bool {
	// Check for explicit watch failures first
	if watcher.watchFailureCount > 0 {
		// If it's been more than 5 minutes since last failure, reset the counter
		if time.Since(watcher.lastWatchFailure) > 5*time.Minute {
			watcher.watchFailureCount = 0
			watcher.controlPlanePause = 0
		} else if watcher.controlPlanePause > 0 {
			return true
		}
	}
	
	// EXPLICIT BACKOFF: Treat ANY recent informer restart as control plane pressure
	if watcher.informerRestarts >= 2 && time.Since(watcher.lastInformerRestart) < 5*time.Minute {
		// Immediate exponential backoff: 2s, 4s, 8s, 16s, 30s (capped)
		backoffSeconds := min(1<<(watcher.informerRestarts-1), 30)
		watcher.controlPlanePause = time.Duration(backoffSeconds) * time.Second
		klog.Warningf("EXPLICIT BACKOFF TRIGGERED: Control plane overloaded (%d restarts), pausing for %v", 
			watcher.informerRestarts, watcher.controlPlanePause)
		return true
	}
	
	return false
}

// recordWatchFailure tracks watch stream failures and calculates exponential backoff
func (watcher *KubeResourceWatcher) recordWatchFailure() {
	watcher.lastWatchFailure = time.Now()
	watcher.watchFailureCount++
	
	// Exponential backoff: 1s, 2s, 4s, 8s, 15s, 30s (capped at 30s)
	backoffSeconds := 1 << min(watcher.watchFailureCount-1, 5) // 2^n but capped
	if backoffSeconds > 30 {
		backoffSeconds = 30
	}
	
	watcher.controlPlanePause = time.Duration(backoffSeconds) * time.Second
	
	klog.Warningf("CONTROL PLANE PRESSURE DETECTED: Recorded watch failure #%d, will pause for %v on next operation", 
		watcher.watchFailureCount, watcher.controlPlanePause)
}

// recordEventProcessingError tracks consecutive event processing errors
func (watcher *KubeResourceWatcher) recordEventProcessingError() {
	watcher.eventProcessingErrors++
	watcher.lastEventError = time.Now()
	
	// If we're getting a lot of consecutive errors, this might indicate control plane pressure
	if watcher.eventProcessingErrors >= 5 && time.Since(watcher.lastEventError) < 2*time.Minute {
		klog.V(2).Infof("CONTROL PLANE PRESSURE DETECTED: %d consecutive event processing errors in recent period", 
			watcher.eventProcessingErrors)
		// Don't record a watch failure here as we may already have recorded it above
		// Just log that we're seeing a pattern
	}
}

// resetEventProcessingErrors resets the error counter after successful processing
func (watcher *KubeResourceWatcher) resetEventProcessingErrors() {
	if watcher.eventProcessingErrors > 0 {
		klog.V(3).Infof("Event processing recovered after %d errors", watcher.eventProcessingErrors)
		watcher.eventProcessingErrors = 0
	}
}

// recordInformerRestart tracks when informers restart (indicating watch stream issues)
func (watcher *KubeResourceWatcher) recordInformerRestart() {
	now := time.Now()
	
	// If it's been more than 5 minutes since the last restart, reset the counter
	if watcher.lastInformerRestart.IsZero() || time.Since(watcher.lastInformerRestart) > 5*time.Minute {
		watcher.informerRestarts = 1
	} else {
		watcher.informerRestarts++
	}
	
	watcher.lastInformerRestart = now
	
	klog.V(2).Infof("Informer restart #%d (watch stream restart detected)", watcher.informerRestarts)
}

// min returns the smaller of two integers (since Go doesn't have built-in min for int)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
