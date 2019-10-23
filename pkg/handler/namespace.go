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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"

	"github.com/fairwindsops/goldilocks/pkg/utils"
	"github.com/fairwindsops/goldilocks/pkg/vpa"
)

// OnNamespaceChanged is a handler that should be called when a namespace chanages.
func OnNamespaceChanged(namespace *corev1.Namespace, event utils.Event) {
	klog.V(7).Infof("Processing namespace: %s", namespace.ObjectMeta.Name)

	switch strings.ToLower(event.EventType) {
	case "delete":
		klog.Info("Nothing to do on namespace deletion. The VPAs will be deleted as part of the ns.")
	case "create", "update":
		klog.Infof("Namespace %s updated. Check the labels.", namespace.ObjectMeta.Name)
		err := vpa.GetInstance().ReconcileNamespace(namespace, false)
		if err != nil {
			klog.Errorf("Error reconciling: %v", err)
		}
	default:
		klog.Infof("Update type %s is not valid, skipping.", event.EventType)
	}
}
