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
	"fmt"
	"github.com/matryer/resync"
	"github.com/mitchellh/go-homedir"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
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

type ConfigContext map[string]string

var kubeClient *ClientInstance
var kubeClientVPA *VPAClientInstance
var clientOnce resync.Once
var clientOnceVPA resync.Once

// configForContext creates a Kubernetes REST client configuration for a given kubeconfig context.
func configForContext(context string) (*rest.Config, error) {
	cfg, err := getConfig(context).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("could not get Kubernetes config for context %q: %s", context, err)
	}
	return cfg, nil
}

// getConfig returns a Kubernetes client config for a given context.
func getConfig(context string) clientcmd.ClientConfig {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	rules.DefaultClientConfig = &clientcmd.DefaultClientConfig

	overrides := &clientcmd.ConfigOverrides{ClusterDefaults: clientcmd.ClusterDefaults}

	if context != "" {
		overrides.CurrentContext = context
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides)
}

// GetContexts returns a Kubernetes interface based on the current configuration
func GetClientCfg(kubeconfigPath string) (*api.Config, error) {
	kubecontexts := make(map[string]string)

	// expand the ~ to the full path
	expandedPath, err := homedir.Expand(kubeconfigPath)
	if err != nil {
		return nil, err
	}

	// Read entire file content, giving us little control but
	// making it very simple. No need to close the file.
	content, err := ioutil.ReadFile(expandedPath)
	if err != nil {
		return nil, err
	}

	// loading all the contexts from the kube config file
	kubeConfigData, err := clientcmd.Load(content)
	if err != nil {
		return nil, err
	}

	// adding the clustername and context name to the map
	for v, c := range kubeConfigData.Contexts {
		kubecontexts[c.Cluster] = v
	}
	return kubeConfigData, nil
}

// GetInstanceWithContext returns a Kubernetes interface based on the current configuration
func GetInstanceWithContext(context string) *ClientInstance {
	clientOnce.Do(func() {
		if kubeClient == nil {
			kubeClient = &ClientInstance{
				Client: getKubeClient(context),
			}
		}
	})
	return kubeClient
}

func ResetInstance() {
	clientOnce.Reset()
	clientOnceVPA.Reset()
	kubeClient = nil
	kubeClientVPA = nil
}

func GetInstance() *ClientInstance {
	clientOnce.Do(func() {
		if kubeClient == nil {
			kubeClient = &ClientInstance{
				Client: getKubeClient(""),
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
				Client: getKubeClientVPA(""),
			}
		}
	})
	return kubeClientVPA
}

// GetVPAInstanceWithContext returns an interface for VPA based on the current configuration
func GetVPAInstanceWithContext(context string) *VPAClientInstance {
	clientOnceVPA.Do(func() {
		if kubeClientVPA == nil {
			kubeClientVPA = &VPAClientInstance{
				Client: getKubeClientVPA(context),
			}
		}
	})
	return kubeClientVPA
}

// getKubeClient creates a Kubernetes config and client for a given kubeconfig context.
func getKubeClient(context string) kubernetes.Interface {
	var kubeConf *rest.Config
	var err error
	if context != "" {
		kubeConf, err = configForContext(context)
		if err != nil {
			klog.Fatalf("Error getting kubeconfig: %v", err)
		}
	} else {
		kubeConf, err = config.GetConfig()
		if err != nil {
			klog.Fatalf("Error getting kubeconfig: %v", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(kubeConf)
	if err != nil {
		klog.Fatalf("Error creating kubernetes client: %v", err)
	}
	return clientset
}

func getKubeClientVPA(context string) autoscalingv1beta2.Interface {
	var kubeConf *rest.Config
	var err error
	if context != "" {
		kubeConf, err = configForContext(context)
		if err != nil {
			klog.Fatalf("Error getting kubeconfig: %v", err)
		}
	} else {
		kubeConf, err = config.GetConfig()
		if err != nil {
			klog.Fatalf("Error getting kubeconfig: %v", err)
		}
	}

	clientset, err := autoscalingv1beta2.NewForConfig(kubeConf)
	if err != nil {
		klog.Fatalf("Error creating kubernetes client: %v", err)
	}
	return clientset
}

// GetNamespace returns a namespace object when given a name.
func GetNamespace(kubeClient *ClientInstance, nsName string) (*corev1.Namespace, error) {

	namespace, err := kubeClient.Client.CoreV1().Namespaces().Get(context.TODO(), nsName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Error getting namespace from name %s: %v", nsName, err)
		return nil, err
	}
	return namespace, nil
}
