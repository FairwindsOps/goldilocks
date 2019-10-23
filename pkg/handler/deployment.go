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

	"github.com/fairwindsops/goldilocks/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/klog"

	"github.com/fairwindsops/goldilocks/pkg/kube"
	"github.com/fairwindsops/goldilocks/pkg/vpa"
)

// OnDeploymentChanged is a handler that should be called when a deployment chanages.
func OnDeploymentChanged(deployment *appsv1.Deployment, event utils.Event) {
	kubeClient := kube.GetInstance()
	namespace, err := kube.GetNamespace(kubeClient, event.Namespace)
	if err != nil {
		klog.Error("Handler got error retrieving namespace object. Breaking.")
		return
	}
	switch strings.ToLower(event.EventType) {
	case "delete":
		klog.V(3).Infof("Deployment %s deleted. Deleting the VPA for it if it had one.", deployment.ObjectMeta.Name)
		err := vpa.GetInstance().ReconcileNamespace(namespace, false)
		if err != nil {
			klog.Errorf("Error reconciling: %v", err)
		}
	case "create", "update":
		klog.V(3).Infof("Deployment %s updated. Reconcile", deployment.ObjectMeta.Name)
		err := vpa.GetInstance().ReconcileNamespace(namespace, false)
		if err != nil {
			klog.Errorf("Error reconciling: %v", err)
		}
	default:
		klog.V(3).Infof("Update type %s is not valid, skipping.", event.EventType)
	}
}
