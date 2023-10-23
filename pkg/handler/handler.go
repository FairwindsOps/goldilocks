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

package handler

import (
	"strings"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/klog/v2"

	"github.com/fairwindsops/goldilocks/pkg/utils"
)

// OnUpdate is a handler that should be called when an object is updated.
// obj is the Kubernetes object that was updated.
// event is the Event metadata representing the update.
func OnUpdate(obj interface{}, event utils.Event) {
	klog.V(10).Infof("Handler got an OnUpdate event of type %s", event.EventType)
	if event.EventType == "delete" {
		onDelete(event)
		return
	}
	switch t := obj.(type) {
	case *corev1.Pod:
		OnPodChanged(obj.(*corev1.Pod), event)
	case *corev1.Namespace:
		OnNamespaceChanged(obj.(*corev1.Namespace), event)
	case *vpav1.VerticalPodAutoscaler:
		OnVPAChanged(obj.(*vpav1.VerticalPodAutoscaler), event)
	case *autoscalingv2.HorizontalPodAutoscaler:
		OnHPAChanged(obj.(*autoscalingv2.HorizontalPodAutoscaler), event)
	default:
		klog.Errorf("Object has unknown type of %T", t)
	}
}

func onDelete(event utils.Event) {
	klog.V(8).Info("OnDelete()")
	switch strings.ToLower(event.ResourceType) {
	case "namespace":
		OnNamespaceChanged(&corev1.Namespace{}, event)
	case "pod":
		OnPodChanged(&corev1.Pod{}, event)
	case "vpa":
		OnVPAChanged(&vpav1.VerticalPodAutoscaler{}, event)
	case "hpa":
		OnHPAChanged(&autoscalingv2.HorizontalPodAutoscaler{}, event)
	default:
		klog.Errorf("object has unknown resource type %s", event.ResourceType)
	}
}
