package kube

import (
	"context"

	"github.com/fairwindsops/controller-utils/pkg/controller"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	v1beta2fake "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned/fake"
	fakedyn "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
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
func GetMockControllerUtilsClient(dynamicClient *DynamicClientInstance) *ControllerUtilsClientInstance {
	kc := ControllerUtilsClientInstance{
		Client: controller.Client{
			Context:    context.TODO(),
			RESTMapper: dynamicClient.RESTMapper,
			Dynamic:    dynamicClient.Client,
		},
	}
	SetControllerUtilsInstance(kc)
	return &kc
}

// GetMockVPAClient returns fake vpa client instance for mocking.
func GetMockDynamicClient() *DynamicClientInstance {
	gvapps := schema.GroupVersion{Group: "apps", Version: "v1"}
	gvcore := schema.GroupVersion{Group: "", Version: "v1"}
	gvbatch := schema.GroupVersion{Group: "batch", Version: "v1"}
	restMapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{gvapps, gvcore, gvbatch})
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
	gvk = gvbatch.WithKind("CronJob")
	restMapper.Add(gvk, meta.RESTScopeNamespace)
	gvk = gvbatch.WithKind("Job")
	restMapper.Add(gvk, meta.RESTScopeNamespace)
	gvrToListKind := map[schema.GroupVersionResource]string{
		{
			Group:    "",
			Version:  "v1",
			Resource: "pods",
		}: "PodList",
		{
			Group:    "",
			Version:  "v1",
			Resource: "namespaces",
		}: "NamespaceList",
		{
			Group:    "apps",
			Version:  "v1",
			Resource: "replicasets",
		}: "ReplicaSetList",
		{
			Group:    "apps",
			Version:  "v1",
			Resource: "deployments",
		}: "DeploymentList",
		{
			Group:    "apps",
			Version:  "v1",
			Resource: "daemonsets",
		}: "DaemonSetList",
		{
			Group:    "apps",
			Version:  "v1",
			Resource: "statefulsets",
		}: "StatefulSetList",
		{
			Group:    "batch",
			Version:  "v1",
			Resource: "cronjobs",
		}: "CronJobList",
		{
			Group:    "batch",
			Version:  "v1",
			Resource: "jobs",
		}: "JobList",
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
