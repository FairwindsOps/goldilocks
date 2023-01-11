package kube

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	v1beta2fake "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned/fake"
	fakedyn "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	"github.com/fairwindsops/controller-utils/pkg/controller"
	"context"
)

// GetMockClient returns a fake client instance for mocking
func GetMockClient() *ClientInstance {
	kc := ClientInstance{
		Client: fake.NewSimpleClientset(),
	}
	SetInstance(kc)
	return &kc
}

// GetMockVPAClient returns fake vpa client instance for mocking.
func GetMockVPAClient() *VPAClientInstance {
	kc := VPAClientInstance{
		Client: v1beta2fake.NewSimpleClientset(),
	}
	SetVPAInstance(kc)
	return &kc
}

// GetMockControllerUtilsClient returns a fake controller client instance for mocking.
func GetMockControllerUtilsClient() *ControllerUtilsClientInstance {
	gvapps := schema.GroupVersion{Group: "apps", Version: "v1"}
	gvcore := schema.GroupVersion{Group: "", Version: "v1"}
	restMapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{gvapps, gvcore})
	fc := fakedyn.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), nil)
	kc := ControllerUtilsClientInstance{
		Client: controller.Client{
			Context: context.TODO(),
			RESTMapper: restMapper,
			Dynamic: fc,
			},
	}
	SetControllerUtilsInstance(kc)
	return &kc
}


// GetMockVPAClient returns fake vpa client instance for mocking.
func GetMockDynamicClient() *DynamicClientInstance {
	gvapps := schema.GroupVersion{Group: "apps", Version: "v1"}
	gvcore := schema.GroupVersion{Group: "", Version: "v1"}
	restMapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{gvapps, gvcore})
	gvk := gvapps.WithKind("Deployment")
	restMapper.Add(gvk, meta.RESTScopeNamespace)
	gvk = gvapps.WithKind("DaemonSet")
	restMapper.Add(gvk, meta.RESTScopeNamespace)
	gvk = gvapps.WithKind("StatefulSet")
	restMapper.Add(gvk, meta.RESTScopeNamespace)
	gvk = gvapps.WithKind("ReplicaSet")
	restMapper.Add(gvk, meta.RESTScopeNamespace)
	gvk = gvcore.WithKind("Pod")
	restMapper.Add(gvk, meta.RESTScopeNamespace)
	gvk = gvcore.WithKind("Namespace")
	restMapper.Add(gvk, meta.RESTScopeRoot)
	gvrToListKind := map[schema.GroupVersionResource]string{
		schema.GroupVersionResource{
			Group:    "",
			Version:  "v1",
			Resource: "pods",
		}: "PodList",
		schema.GroupVersionResource{
			Group:    "",
			Version:  "v1",
			Resource: "namespaces",
		}: "NamespaceList",
		schema.GroupVersionResource{
			Group:    "apps",
			Version:  "v1",
			Resource: "replicasets",
		}: "ReplicaSetList",
		schema.GroupVersionResource{
			Group:    "apps",
			Version:  "v1",
			Resource: "deployments",
		}: "DeploymentList",
		schema.GroupVersionResource{
			Group:    "apps",
			Version:  "v1",
			Resource: "daemonsets",
		}: "DaemonSetList",
		schema.GroupVersionResource{
			Group:    "apps",
			Version:  "v1",
			Resource: "statefulsets",
		}: "StatefulSetList",
	}
	fc := fakedyn.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind)
	kc := DynamicClientInstance{
		Client:     fc,
		RESTMapper: restMapper,
	}
	SetDynamicInstance(kc)
	return &kc
}

// SetInstance allows the user to set the kubeClient singleton
func SetInstance(kc ClientInstance) {
	kubeClient = &kc
}

// SetVPAInstance sets the kubeClient for VPA
func SetVPAInstance(kc VPAClientInstance) {
	kubeClientVPA = &kc
}

// SetVPAInstance sets the kubeClient for VPA
func SetDynamicInstance(kc DynamicClientInstance) {
	dynamicClient = &kc
}

// SetControllerUtilsInstance sets a kubeClient for Controller
func SetControllerUtilsInstance(kc ControllerUtilsClientInstance) {
	controllerUtilsClient = &kc
}
