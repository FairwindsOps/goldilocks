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

package vpa

import (
	"testing"

	"github.com/fairwindsops/goldilocks/pkg/kube"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2"
)

func setupVPAForTests() {
	kubeClient := kube.GetMockClient()
	vpaClient := kube.GetMockVPAClient()
	testVPAReconciler := SetInstance(kubeClient, vpaClient)
	testVPAReconciler.OnByDefault = false
	testVPAReconciler.IncludeNamespaces = []string{}
	testVPAReconciler.ExcludeNamespaces = []string{}
}

func Test_createVPADryRun(t *testing.T) {
	setupVPAForTests()
	VPAClient := GetInstance().VPAClient

	// First test the dryrun
	err := createVPA(VPAClient, "testing", "test-vpa", true)
	assert.NoError(t, err)
	_, err = VPAClient.Client.AutoscalingV1beta2().VerticalPodAutoscalers("testing").Get("test-vpa", metav1.GetOptions{})
	assert.EqualError(t, err, "verticalpodautoscalers.autoscaling.k8s.io \"test-vpa\" not found")

	// Now actually create and compare
	errCreate := createVPA(VPAClient, "testing", "test-vpa", false)
	newVPA, _ := VPAClient.Client.AutoscalingV1beta2().VerticalPodAutoscalers("testing").Get("test-vpa", metav1.GetOptions{})
	assert.NoError(t, errCreate)
	assert.EqualValues(t, testVPA, newVPA)
}

func Test_deleteVPA(t *testing.T) {
	setupVPAForTests()
	VPAClient := GetInstance().VPAClient

	_, err := VPAClient.Client.AutoscalingV1beta2().VerticalPodAutoscalers("testing").Create(testVPA)
	assert.NoError(t, err)

	// Test deletion dryrun
	errDeleteDryRun := deleteVPA(VPAClient, "testing", "test-vpa", true)
	assert.NoError(t, errDeleteDryRun)
	oldVPA, _ := VPAClient.Client.AutoscalingV1beta2().VerticalPodAutoscalers("testing").Get("test-vpa", metav1.GetOptions{})
	assert.EqualValues(t, testVPA, oldVPA)

	// Test actual deletion
	errDelete := deleteVPA(VPAClient, "testing", "test-vpa", false)
	assert.NoError(t, errDelete)
	_, errNotFound := VPAClient.Client.AutoscalingV1beta2().VerticalPodAutoscalers("testing").Get("test-vpa", metav1.GetOptions{})
	assert.EqualError(t, errNotFound, "verticalpodautoscalers.autoscaling.k8s.io \"test-vpa\" not found")
}

func Test_listVPA(t *testing.T) {
	setupVPAForTests()
	VPAClient := GetInstance().VPAClient

	_ = createVPA(VPAClient, "ns", "test1", false)
	_ = createVPA(VPAClient, "ns", "test2", false)
	_ = createVPA(VPAClient, "ns2", "test3", false)

	vpaList1 := listVPA(VPAClient, "ns")
	assert.EqualValues(t, vpaList1, []string{"test1", "test2"})

	vpaList2 := listVPA(VPAClient, "")
	assert.EqualValues(t, vpaList2, []string{"test1", "test2", "test3"})

	var expected []string
	vpaList3 := listVPA(VPAClient, "nonexistent")
	assert.EqualValues(t, vpaList3, expected)
}

func Test_checkNamespaceLabel(t *testing.T) {
	tests := []struct {
		name      string
		namespace *corev1.Namespace
		want      bool
	}{
		{
			name:      "Labeled correctly",
			namespace: nsLabeledTrue,
			want:      true,
		},
		{
			name:      "Labeled Incorrectly",
			namespace: nsLabeledFalse,
			want:      false,
		},
		{
			name:      "Not Labeled",
			namespace: nsNotLabeled,
			want:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetInstance().checkNamespaceLabel(tt.namespace)
			assert.Equal(t, got, tt.want)
		})
	}
}

func Test_checkNamespaceLists(t *testing.T) {
	setupVPAForTests()
	vpaReconciler := GetInstance()

	vpaReconciler.OnByDefault = false
	vpaReconciler.IncludeNamespaces = []string{}
	vpaReconciler.ExcludeNamespaces = []string{}
	got := vpaReconciler.checkNamespaceLabel(nsNotLabeled)
	assert.Equal(t, false, got)

	vpaReconciler.OnByDefault = true
	vpaReconciler.IncludeNamespaces = []string{}
	vpaReconciler.ExcludeNamespaces = []string{}
	got = vpaReconciler.checkNamespaceLabel(nsNotLabeled)
	assert.Equal(t, true, got)

	vpaReconciler.OnByDefault = false
	vpaReconciler.IncludeNamespaces = []string{nsNotLabeled.ObjectMeta.Name}
	vpaReconciler.ExcludeNamespaces = []string{}
	got = vpaReconciler.checkNamespaceLabel(nsNotLabeled)
	assert.Equal(t, true, got)

	vpaReconciler.OnByDefault = true
	vpaReconciler.IncludeNamespaces = []string{}
	vpaReconciler.ExcludeNamespaces = []string{nsNotLabeled.ObjectMeta.Name}
	got = vpaReconciler.checkNamespaceLabel(nsNotLabeled)
	assert.Equal(t, false, got)

	// Labels take precedence over CLI options
	vpaReconciler.OnByDefault = true
	vpaReconciler.IncludeNamespaces = []string{}
	vpaReconciler.ExcludeNamespaces = []string{}
	got = vpaReconciler.checkNamespaceLabel(nsLabeledFalse)
	assert.Equal(t, false, got)

	vpaReconciler.OnByDefault = false
	vpaReconciler.IncludeNamespaces = []string{nsLabeledFalse.ObjectMeta.Name}
	vpaReconciler.ExcludeNamespaces = []string{}
	got = vpaReconciler.checkNamespaceLabel(nsLabeledFalse)
	assert.Equal(t, false, got)

	vpaReconciler.OnByDefault = false
	vpaReconciler.IncludeNamespaces = []string{}
	vpaReconciler.ExcludeNamespaces = []string{nsLabeledTrue.ObjectMeta.Name}
	got = vpaReconciler.checkNamespaceLabel(nsLabeledTrue)
	assert.Equal(t, true, got)
}

func Test_checkDeploymentLabels(t *testing.T) {
	tests := []struct {
		name       string
		deployment *appsv1.Deployment
		want       bool
		wantErr    bool
		err        string
	}{
		{
			name: "Labeled correctly",
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "labeled-true",
					Labels: map[string]string{
						"goldilocks.fairwinds.com/enabled": "True",
					},
				},
			},
			want:    true,
			wantErr: false,
			err:     "",
		},
		{
			name: "Labeled Incorrectly",
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "labeled-false",
					Labels: map[string]string{
						"goldilocks.fairwinds.com/enabled": "false",
					},
				},
			},
			want:    false,
			wantErr: false,
			err:     "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetInstance().checkDeploymentLabels(tt.deployment)
			if tt.wantErr {
				assert.EqualError(t, err, tt.err)
			} else {
				assert.Equal(t, got, tt.want)
			}
		})
	}
}

func Test_ReconcileNamespaceNoLabels(t *testing.T) {
	setupVPAForTests()
	VPAClient := GetInstance().VPAClient
	KubeClient := GetInstance().KubeClient

	_, err := KubeClient.Client.CoreV1().Namespaces().Create(nsLabeledFalse)
	assert.NoError(t, err)
	nsName := nsLabeledFalse.ObjectMeta.Name

	_, err = KubeClient.Client.AppsV1().Deployments(nsName).Create(testDeployment)
	assert.NoError(t, err)

	// False labels should generate 0 vpa objects
	err = GetInstance().ReconcileNamespace(nsLabeledFalse, false)
	assert.NoError(t, err)

	vpaList, err := VPAClient.Client.AutoscalingV1beta2().VerticalPodAutoscalers(nsName).List(metav1.ListOptions{})
	assert.Equal(t, 0, len(vpaList.Items))
	assert.NoError(t, err)
	assert.EqualValues(t, vpaList, &v1beta2.VerticalPodAutoscalerList{})
}

func Test_ReconcileNamespaceWithLabels(t *testing.T) {
	setupVPAForTests()
	VPAClient := GetInstance().VPAClient
	KubeClient := GetInstance().KubeClient

	_, err := KubeClient.Client.CoreV1().Namespaces().Create(nsLabeledTrue)
	assert.NoError(t, err)
	nsName := nsLabeledTrue.ObjectMeta.Name

	_, err = KubeClient.Client.AppsV1().Deployments(nsName).Create(testDeployment)
	assert.NoError(t, err)

	// This should create a single VPA
	err = GetInstance().ReconcileNamespace(nsLabeledTrue, false)
	assert.NoError(t, err)

	vpaList, err := VPAClient.Client.AutoscalingV1beta2().VerticalPodAutoscalers(nsName).List(metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(vpaList.Items))
	assert.Equal(t, "test-deploy", vpaList.Items[0].ObjectMeta.Name)
}

func Test_ReconcileNamespaceDeleteDeployment(t *testing.T) {
	setupVPAForTests()
	VPAClient := GetInstance().VPAClient
	KubeClient := GetInstance().KubeClient

	_, err := KubeClient.Client.CoreV1().Namespaces().Create(nsLabeledTrue)
	assert.NoError(t, err)
	nsName := nsLabeledTrue.ObjectMeta.Name

	// Create deploy, reconcile, delete deploy, reconcile
	_, err = KubeClient.Client.AppsV1().Deployments(nsName).Create(testDeployment)
	assert.NoError(t, err)
	err = GetInstance().ReconcileNamespace(nsLabeledTrue, false)
	assert.NoError(t, err)
	err = KubeClient.Client.AppsV1().Deployments(nsName).Delete(testDeployment.ObjectMeta.Name, &metav1.DeleteOptions{})
	assert.NoError(t, err)
	err = GetInstance().ReconcileNamespace(nsLabeledTrue, false)
	assert.NoError(t, err)

	// No VPA objects left after deleted deployment
	vpaList, err := VPAClient.Client.AutoscalingV1beta2().VerticalPodAutoscalers(nsName).List(metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(vpaList.Items))
	assert.EqualValues(t, vpaList, &v1beta2.VerticalPodAutoscalerList{})

}

func Test_ReconcileNamespaceRemoveLabel(t *testing.T) {
	setupVPAForTests()
	VPAClient := GetInstance().VPAClient
	KubeClient := GetInstance().KubeClient

	// Create a properly labeled namespace
	_, err := KubeClient.Client.CoreV1().Namespaces().Create(nsLabeledTrue)
	assert.NoError(t, err)
	nsName := nsLabeledTrue.ObjectMeta.Name

	// Create a deployment in the namespace and reconcile
	_, err = KubeClient.Client.AppsV1().Deployments(nsName).Create(testDeployment)
	assert.NoError(t, err)
	err = GetInstance().ReconcileNamespace(nsLabeledTrue, false)
	assert.NoError(t, err)

	// Update the namespace labels to be false and reconcile
	updatedNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
			Labels: map[string]string{
				"goldilocks.fairwinds.com/enabled": "false",
			},
		},
	}
	_, err = KubeClient.Client.CoreV1().Namespaces().Update(updatedNS)
	assert.NoError(t, err)
	err = GetInstance().ReconcileNamespace(updatedNS, false)
	assert.NoError(t, err)

	// There should be zero vpa objects
	vpaList, err := VPAClient.Client.AutoscalingV1beta2().VerticalPodAutoscalers(nsName).List(metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(vpaList.Items))
	assert.EqualValues(t, vpaList, &v1beta2.VerticalPodAutoscalerList{})
}
