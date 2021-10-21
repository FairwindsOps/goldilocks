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

	"github.com/davecgh/go-spew/spew"
	"github.com/fairwindsops/goldilocks/pkg/kube"
	"github.com/fairwindsops/goldilocks/pkg/utils"
	"github.com/fairwindsops/goldilocks/pkg/vpa"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

func OnPodChanged(pod *corev1.Pod, event utils.Event) {
	kubeClient := kube.GetInstance()
	namespace, err := kube.GetNamespace(kubeClient, event.Namespace)
	if err != nil {
		klog.Error("handler got error retrieving namespace object. breaking.")
		klog.V(5).Info("dumping out event struct")
		klog.V(5).Info(spew.Sdump(event))
		return
	}
	switch strings.ToLower(event.EventType) {
	case "delete":
		klog.V(3).Infof("Pod %s deleted. Deleting the VPA for it if it had one.", pod.ObjectMeta.Name)
		err := vpa.GetInstance().ReconcileNamespace(namespace)
		if err != nil {
			klog.Errorf("Error reconciling: %v", err)
		}
	case "create", "update":
		klog.V(3).Infof("Pod %s updated. Reconcile", pod.ObjectMeta.Name)
		err := vpa.GetInstance().ReconcileNamespace(namespace)
		if err != nil {
			klog.Errorf("Error reconciling: %v", err)
		}
	default:
		klog.V(3).Infof("Update type %s is not valid, skipping.", event.EventType)
	}
}
