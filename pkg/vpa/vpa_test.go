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
	"context"
	"testing"

	"github.com/fairwindsops/goldilocks/pkg/kube"
	"github.com/fairwindsops/goldilocks/pkg/utils"
	"github.com/stretchr/testify/assert"

	appsv1 "k8s.io/api/apps/v1"

	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
)

func setupVPAForTests() {
	kubeClient := kube.GetMockClient()
	vpaClient := kube.GetMockVPAClient()
	testVPAReconciler := SetInstance(kubeClient, vpaClient)
	testVPAReconciler.OnByDefault = false
	testVPAReconciler.IncludeNamespaces = []string{}
	testVPAReconciler.ExcludeNamespaces = []string{}
}

func Test_vpaUpdateModeForNamespace(t *testing.T) {
	setupVPAForTests()

	tests := []struct {
		name       string
		ns         *corev1.Namespace
		explicit   bool
		updateMode vpav1.UpdateMode
	}{
		{
			name:       "unlabeled (default)",
			ns:         nsNotLabeled,
			explicit:   false,
			updateMode: vpav1.UpdateModeOff,
		},
		{
			name:       "labeled: enabled=false",
			ns:         nsLabeledFalse,
			explicit:   false,
			updateMode: vpav1.UpdateModeOff,
		},
		{
			name:       "labled: enabled=true",
			ns:         nsLabeledTrue,
			explicit:   false,
			updateMode: vpav1.UpdateModeOff,
		},
		{
			name:       "labled: enabled=true, vpa-update-mode=off",
			ns:         nsLabeledTrueUpdateModeOff,
			explicit:   true,
			updateMode: vpav1.UpdateModeOff,
		},
		{
			name:       "labled: enabled=true, vpa-update-mode=auto",
			ns:         nsLabeledTrueUpdateModeAuto,
			explicit:   true,
			updateMode: vpav1.UpdateModeAuto,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			want := test.updateMode
			got, explicit := vpaUpdateModeForResource(test.ns)
			assert.Equal(t, want, *got)
			assert.Equal(t, test.explicit, explicit)
		})
	}
}

func Test_getVPAObject(t *testing.T) {
	setupVPAForTests()
	rec := GetInstance()

	tests := []struct {
		name       string
		ns         *corev1.Namespace
		updateMode vpav1.UpdateMode
		vpa        *vpav1.VerticalPodAutoscaler
	}{
		{
			name:       "example-vpa",
			ns:         nsLabeledTrueUpdateModeAuto,
			updateMode: vpav1.UpdateModeAuto,
			vpa:        nil,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			mode, _ := vpaUpdateModeForResource(test.ns)
			vpa := rec.getVPAObject(test.vpa, test.ns, "test-vpa", mode)

			// expected ObjectMeta
			assert.Equal(t, "test-vpa", vpa.Name)
			assert.Equal(t, test.ns.Name, vpa.Namespace)
			assert.Equal(t, utils.VPALabels, vpa.Labels)

			// expected .spec.target
			// workload target matches the vpa name
			assert.Equal(t, vpa.Name, vpa.Spec.TargetRef.Name)
			// update mode is correct for the namespace
			assert.Equal(t, test.updateMode, *vpa.Spec.UpdatePolicy.UpdateMode)
		})
	}
}

func Test_createVPA(t *testing.T) {
	setupVPAForTests()
	VPAClient := GetInstance().VPAClient

	// First test the dryrun
	rec := GetInstance()
	rec.DryRun = true

	updateMode, _ := vpaUpdateModeForResource(nsTesting)
	testVPA := rec.getVPAObject(nil, nsTesting, "test-vpa", updateMode)

	err := rec.createVPA(testVPA)
	assert.NoError(t, err)
	_, err = VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsTesting.Name).Get(context.TODO(), "test-vpa", metav1.GetOptions{})
	assert.EqualError(t, err, "verticalpodautoscalers.autoscaling.k8s.io \"test-vpa\" not found")

	// Now actually create and compare
	rec.DryRun = false
	errCreate := rec.createVPA(testVPA)
	newVPA, _ := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsTesting.Name).Get(context.TODO(), "test-vpa", metav1.GetOptions{})
	assert.NoError(t, errCreate)
	assert.EqualValues(t, &testVPA, newVPA)
}

func Test_deleteVPA(t *testing.T) {
	setupVPAForTests()
	VPAClient := GetInstance().VPAClient

	// First test the dryrun
	rec := GetInstance()
	rec.DryRun = true

	updateMode, _ := vpaUpdateModeForResource(nsTesting)
	testVPA := rec.getVPAObject(nil, nsTesting, "test-vpa", updateMode)
	_, err := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsTesting.Name).Create(context.TODO(), &testVPA, metav1.CreateOptions{})
	assert.NoError(t, err)

	errDeleteDryRun := rec.deleteVPA(testVPA)
	assert.NoError(t, errDeleteDryRun)
	oldVPA, _ := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsTesting.Name).Get(context.TODO(), "test-vpa", metav1.GetOptions{})
	assert.EqualValues(t, &testVPA, oldVPA)

	// Test actual deletion
	rec.DryRun = false
	errDelete := rec.deleteVPA(testVPA)
	assert.NoError(t, errDelete)
	_, errNotFound := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers("testing").Get(context.TODO(), "test-vpa", metav1.GetOptions{})
	assert.EqualError(t, errNotFound, "verticalpodautoscalers.autoscaling.k8s.io \"test-vpa\" not found")
}

func Test_updateVPA(t *testing.T) {
	setupVPAForTests()
	VPAClient := GetInstance().VPAClient

	// First test the dryrun
	rec := GetInstance()
	rec.DryRun = true

	testNS := nsTesting.DeepCopy()
	testNS.Labels["goldilocks.fairwinds.com/vpa-update-mode"] = "off"

	updateMode, _ := vpaUpdateModeForResource(testNS)
	testVPA := rec.getVPAObject(nil, testNS, "test-vpa", updateMode)
	_, err := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(testNS.Name).Create(context.TODO(), &testVPA, metav1.CreateOptions{})
	assert.NoError(t, err)

	// dry run
	errUpdateDryRun := rec.updateVPA(testVPA)
	assert.NoError(t, errUpdateDryRun)
	currVPA, _ := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(testNS.Name).Get(context.TODO(), "test-vpa", metav1.GetOptions{})
	assert.EqualValues(t, &testVPA, currVPA)

	// live update
	rec.DryRun = false
	errUpdate := rec.updateVPA(testVPA)
	assert.NoError(t, errUpdate)
	currVPA, _ = VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(testNS.Name).Get(context.TODO(), "test-vpa", metav1.GetOptions{})
	// no change between create and update
	assert.EqualValues(t, &testVPA, currVPA)

	// change the update mode
	testNS.Labels["goldilocks.fairwinds.com/vpa-update-mode"] = "auto"
	updateMode, _ = vpaUpdateModeForResource(testNS)
	newVPA := rec.getVPAObject(nil, testNS, "test-vpa", updateMode)

	errUpdate2 := rec.updateVPA(newVPA)
	assert.NoError(t, errUpdate2)
	currVPA, _ = VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(testNS.Name).Get(context.TODO(), "test-vpa", metav1.GetOptions{})
	// no change between create and update
	assert.NotEqual(t, &testVPA, currVPA)
	// check that the update mode changed
	assert.Equal(t, updateModeAuto, *currVPA.Spec.UpdatePolicy.UpdateMode)
}

func Test_listVPA(t *testing.T) {
	setupVPAForTests()
	rec := GetInstance()

	// test namespaces
	testNS1 := nsTesting.DeepCopy()
	testNS1.Name = "ns1"
	testNS1.Namespace = "ns1"
	testNS2 := nsTesting.DeepCopy()
	testNS2.Name = "ns2"
	testNS2.Namespace = "ns2"

	// test vpas
	updateMode1, _ := vpaUpdateModeForResource(testNS1)
	updateMode2, _ := vpaUpdateModeForResource(testNS2)
	vpa1 := rec.getVPAObject(nil, testNS1, "test1", updateMode1)
	vpa2 := rec.getVPAObject(nil, testNS1, "test2", updateMode1)
	vpa3 := rec.getVPAObject(nil, testNS2, "test3", updateMode2)

	// create vpas
	_ = rec.createVPA(vpa1)
	_ = rec.createVPA(vpa2)
	_ = rec.createVPA(vpa3)

	// list ns1
	vpaList1, err := rec.listVPAs("ns1")
	assert.NoError(t, err)
	assert.NotEmpty(t, vpaList1)
	assert.EqualValues(t, vpaList1[0].Name, "test1")
	assert.EqualValues(t, vpaList1[1].Name, "test2")

	// list all
	vpaList2, err := rec.listVPAs("")
	assert.NoError(t, err)
	assert.NotEmpty(t, vpaList2)
	assert.EqualValues(t, vpaList2[0].Name, "test1")
	assert.EqualValues(t, vpaList2[1].Name, "test2")
	assert.EqualValues(t, vpaList2[2].Name, "test3")

	// list dne
	vpaList3, err := rec.listVPAs("nonexistent")
	assert.NoError(t, err)
	assert.Empty(t, vpaList3)
}

func Test_namespaceIsManaged(t *testing.T) {
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
			got := GetInstance().namespaceIsManaged(tt.namespace)
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
	got := vpaReconciler.namespaceIsManaged(nsNotLabeled)
	assert.Equal(t, false, got)

	vpaReconciler.OnByDefault = true
	vpaReconciler.IncludeNamespaces = []string{}
	vpaReconciler.ExcludeNamespaces = []string{}
	got = vpaReconciler.namespaceIsManaged(nsNotLabeled)
	assert.Equal(t, true, got)

	vpaReconciler.OnByDefault = false
	vpaReconciler.IncludeNamespaces = []string{nsNotLabeled.ObjectMeta.Name}
	vpaReconciler.ExcludeNamespaces = []string{}
	got = vpaReconciler.namespaceIsManaged(nsNotLabeled)
	assert.Equal(t, true, got)

	vpaReconciler.OnByDefault = true
	vpaReconciler.IncludeNamespaces = []string{}
	vpaReconciler.ExcludeNamespaces = []string{nsNotLabeled.ObjectMeta.Name}
	got = vpaReconciler.namespaceIsManaged(nsNotLabeled)
	assert.Equal(t, false, got)

	// Labels take precedence over CLI options
	vpaReconciler.OnByDefault = true
	vpaReconciler.IncludeNamespaces = []string{}
	vpaReconciler.ExcludeNamespaces = []string{}
	got = vpaReconciler.namespaceIsManaged(nsLabeledFalse)
	assert.Equal(t, false, got)

	vpaReconciler.OnByDefault = false
	vpaReconciler.IncludeNamespaces = []string{nsLabeledFalse.ObjectMeta.Name}
	vpaReconciler.ExcludeNamespaces = []string{}
	got = vpaReconciler.namespaceIsManaged(nsLabeledFalse)
	assert.Equal(t, false, got)

	vpaReconciler.OnByDefault = false
	vpaReconciler.IncludeNamespaces = []string{}
	vpaReconciler.ExcludeNamespaces = []string{nsLabeledTrue.ObjectMeta.Name}
	got = vpaReconciler.namespaceIsManaged(nsLabeledTrue)
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

	_, err := KubeClient.Client.CoreV1().Namespaces().Create(context.TODO(), nsLabeledFalse, metav1.CreateOptions{})
	assert.NoError(t, err)
	nsName := nsLabeledFalse.ObjectMeta.Name

	_, err = KubeClient.Client.AppsV1().Deployments(nsName).Create(context.TODO(), testDeployment, metav1.CreateOptions{})
	assert.NoError(t, err)

	// False labels should generate 0 vpa objects
	err = GetInstance().ReconcileNamespace(nsLabeledFalse)
	assert.NoError(t, err)

	vpaList, err := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsName).List(context.TODO(), metav1.ListOptions{})
	assert.Equal(t, 0, len(vpaList.Items))
	assert.NoError(t, err)
	assert.EqualValues(t, vpaList, &vpav1.VerticalPodAutoscalerList{})
}

func Test_ReconcileNamespaceWithLabels(t *testing.T) {
	setupVPAForTests()
	VPAClient := GetInstance().VPAClient
	KubeClient := GetInstance().KubeClient

	_, err := KubeClient.Client.CoreV1().Namespaces().Create(context.TODO(), nsLabeledTrue, metav1.CreateOptions{})
	assert.NoError(t, err)
	nsName := nsLabeledTrue.ObjectMeta.Name

	_, err = KubeClient.Client.AppsV1().Deployments(nsName).Create(context.TODO(), testDeployment, metav1.CreateOptions{})
	assert.NoError(t, err)

	// This should create a single VPA
	err = GetInstance().ReconcileNamespace(nsLabeledTrue)
	assert.NoError(t, err)

	vpaList, err := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsName).List(context.TODO(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(vpaList.Items))
	assert.Equal(t, "test-deploy", vpaList.Items[0].ObjectMeta.Name)
}

func Test_ReconcileNamespaceDeleteDeployment(t *testing.T) {
	setupVPAForTests()
	VPAClient := GetInstance().VPAClient
	KubeClient := GetInstance().KubeClient

	_, err := KubeClient.Client.CoreV1().Namespaces().Create(context.TODO(), nsLabeledTrue, metav1.CreateOptions{})
	assert.NoError(t, err)
	nsName := nsLabeledTrue.ObjectMeta.Name

	// Create deploy, reconcile, delete deploy, reconcile
	_, err = KubeClient.Client.AppsV1().Deployments(nsName).Create(context.TODO(), testDeployment, metav1.CreateOptions{})
	assert.NoError(t, err)
	err = GetInstance().ReconcileNamespace(nsLabeledTrue)
	assert.NoError(t, err)
	err = KubeClient.Client.AppsV1().Deployments(nsName).Delete(context.TODO(), testDeployment.ObjectMeta.Name, metav1.DeleteOptions{})
	assert.NoError(t, err)
	err = GetInstance().ReconcileNamespace(nsLabeledTrue)
	assert.NoError(t, err)

	// No VPA objects left after deleted deployment
	vpaList, err := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsName).List(context.TODO(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(vpaList.Items))
	assert.EqualValues(t, vpaList, &vpav1.VerticalPodAutoscalerList{})

}

func Test_ReconcileNamespaceRemoveLabel(t *testing.T) {
	setupVPAForTests()
	VPAClient := GetInstance().VPAClient
	KubeClient := GetInstance().KubeClient

	// Create a properly labeled namespace
	_, err := KubeClient.Client.CoreV1().Namespaces().Create(context.TODO(), nsLabeledTrue, metav1.CreateOptions{})
	assert.NoError(t, err)
	nsName := nsLabeledTrue.ObjectMeta.Name

	// Create a deployment in the namespace and reconcile
	_, err = KubeClient.Client.AppsV1().Deployments(nsName).Create(context.TODO(), testDeployment, metav1.CreateOptions{})
	assert.NoError(t, err)
	err = GetInstance().ReconcileNamespace(nsLabeledTrue)
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
	_, err = KubeClient.Client.CoreV1().Namespaces().Update(context.TODO(), updatedNS, metav1.UpdateOptions{})
	assert.NoError(t, err)
	err = GetInstance().ReconcileNamespace(updatedNS)
	assert.NoError(t, err)

	// There should be zero vpa objects
	vpaList, err := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsName).List(context.TODO(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(vpaList.Items))
	assert.EqualValues(t, vpaList, &vpav1.VerticalPodAutoscalerList{})
}

func Test_ReconcileNamespace_ExcludeDeploymentAnnotation(t *testing.T) {
	setupVPAForTests()
	VPAClient := GetInstance().VPAClient
	KubeClient := GetInstance().KubeClient

	// Create a properly labeled namespace
	_, err := KubeClient.Client.CoreV1().Namespaces().Create(context.TODO(), nsLabeledTrueUpdateModeAuto, metav1.CreateOptions{})
	assert.NoError(t, err)
	nsName := nsLabeledTrue.ObjectMeta.Name

	// Create an excluded deployment in the namespace and reconcile
	_, err = KubeClient.Client.AppsV1().Deployments(nsName).Create(context.TODO(), testDeploymentExcluded, metav1.CreateOptions{})
	assert.NoError(t, err)
	err = GetInstance().ReconcileNamespace(nsLabeledTrue)
	assert.NoError(t, err)

	// There should be one vpa object with UpdateModeOff
	vpaList, err := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsName).List(context.TODO(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(vpaList.Items))
	assert.EqualValues(t, *vpaList.Items[0].Spec.UpdatePolicy.UpdateMode, vpav1.UpdateModeOff)
}

func Test_ReconcileNamespace_ChangeUpdateMode(t *testing.T) {
	setupVPAForTests()
	VPAClient := GetInstance().VPAClient
	KubeClient := GetInstance().KubeClient

	// Create a properly labeled namespace
	_, err := KubeClient.Client.CoreV1().Namespaces().Create(context.TODO(), nsLabeledTrue, metav1.CreateOptions{})
	assert.NoError(t, err)
	nsName := nsLabeledTrue.ObjectMeta.Name

	// Create a deployment in the namespace and reconcile
	_, err = KubeClient.Client.AppsV1().Deployments(nsName).Create(context.TODO(), testDeployment, metav1.CreateOptions{})
	assert.NoError(t, err)
	err = GetInstance().ReconcileNamespace(nsLabeledTrue)
	assert.NoError(t, err)

	// There should be one vpa object with updatemode "off"
	vpaList, err := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsName).List(context.TODO(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(vpaList.Items))
	assert.EqualValues(t, *vpaList.Items[0].Spec.UpdatePolicy.UpdateMode, vpav1.UpdateModeOff)

	// Update the namespace labels to be be vpa-update-mode=auto and reconcile
	updatedNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
			Labels: map[string]string{
				"goldilocks.fairwinds.com/enabled":         "True",
				"goldilocks.fairwinds.com/vpa-update-mode": "auto",
			},
		},
	}
	_, err = KubeClient.Client.CoreV1().Namespaces().Update(context.TODO(), updatedNS, metav1.UpdateOptions{})
	assert.NoError(t, err)
	err = GetInstance().ReconcileNamespace(updatedNS)
	assert.NoError(t, err)

	// There should be one vpa object with updatemode "auto"
	vpaList1, err := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsName).List(context.TODO(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(vpaList1.Items))
	assert.EqualValues(t, *vpaList1.Items[0].Spec.UpdatePolicy.UpdateMode, vpav1.UpdateModeAuto)
}
