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
	"sync"

	"k8s.io/client-go/kubernetes"
	// Empty imports needed for supported auth methods in kubeconfig. See client-go documentation
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	autoscalingv1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned"
)

// ClientInstance is a wrapper around the kubernetes interface for testing purposes
type ClientInstance struct {
	Client kubernetes.Interface
}

// VPAClientInstance is a wrapper around the autoscaling interface for testing purposes
type VPAClientInstance struct {
	Client autoscalingv1beta2.Interface
}

var kubeClient *ClientInstance
var kubeClientVPA *VPAClientInstance
var clientOnce sync.Once
var clientOnceVPA sync.Once

// GetInstance returns a Kubernetes interface based on the current configuration
func GetInstance() *ClientInstance {
	clientOnce.Do(func() {
		if kubeClient == nil {
			kubeClient = &ClientInstance{
				Client: getKubeClient(),
			}
		}
	})
	return kubeClient
}

// GetVPAInstance returns an interface for VPA based on the current configuration
func GetVPAInstance() *VPAClientInstance {
	clientOnceVPA.Do(func() {
		if kubeClientVPA == nil {
			kubeClientVPA = &VPAClientInstance{
				Client: getKubeClientVPA(),
			}
		}
	})
	return kubeClientVPA
}

func getKubeClient() kubernetes.Interface {
	kubeConf, err := config.GetConfig()
	if err != nil {
		klog.Fatalf("Error getting kubeconfig: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(kubeConf)
	if err != nil {
		klog.Fatalf("Error creating kubernetes client: %v", err)
	}
	return clientset
}

func getKubeClientVPA() autoscalingv1beta2.Interface {
	kubeConf, err := config.GetConfig()
	if err != nil {
		klog.Fatalf("Error getting kubeconfig: %v", err)
	}
	clientset, err := autoscalingv1beta2.NewForConfig(kubeConf)
	if err != nil {
		klog.Fatalf("Error creating kubernetes client: %v", err)
	}
	return clientset
}

// GetNamespace returns a namespace object when given a name.
func GetNamespace(kubeClient *ClientInstance, nsName string) (*corev1.Namespace, error) {

	namespace, err := kubeClient.Client.CoreV1().Namespaces().Get(nsName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Error getting namespace from name %s: %v", nsName, err)
		return nil, err
	}
	return namespace, nil
}
