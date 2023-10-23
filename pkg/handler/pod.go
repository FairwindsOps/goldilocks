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
	"github.com/davecgh/go-spew/spew"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	"github.com/fairwindsops/goldilocks/pkg/kube"
	"github.com/fairwindsops/goldilocks/pkg/utils"
	"github.com/fairwindsops/goldilocks/pkg/vpa"
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

	klog.V(3).Infof("Pod %s/%s changed with %q. Reconciling namespace.", pod.Namespace, pod.Name, event.EventType)
	err = vpa.GetInstance().ReconcileNamespace(namespace)
	if err != nil {
		klog.Errorf("Error reconciling: %v", err)
	}
}
