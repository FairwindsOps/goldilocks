package kube

import (
	v1beta2fake "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned/fake"
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

// SetInstance allows the user to set the kubeClient singleton
func SetInstance(kc ClientInstance) {
	kubeClient = &kc
}

// SetVPAInstance sets the kubeClient for VPA
func SetVPAInstance(kc VPAClientInstance) {
	kubeClientVPA = &kc
}
