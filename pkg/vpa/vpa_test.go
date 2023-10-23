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
	"strings"
	"testing"

	"github.com/fairwindsops/goldilocks/pkg/kube"
	"github.com/fairwindsops/goldilocks/pkg/utils"
	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
)

func setupVPAForTests(t *testing.T) {
	kubeClient := kube.GetMockClient()
	vpaClient := kube.GetMockVPAClient()
	dynamicClient := kube.GetMockDynamicClient()
	controllerUtilsClient := kube.GetMockControllerUtilsClient(dynamicClient)
	testVPAReconciler := SetInstance(kubeClient, vpaClient, dynamicClient, controllerUtilsClient)
	testVPAReconciler.OnByDefault = false
	testVPAReconciler.IncludeNamespaces = []string{}
	testVPAReconciler.ExcludeNamespaces = []string{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(nsLabeledTrueUnstructured.Object, &nsLabeledTrue)
	assert.NoError(t, err)
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(nsLabeledFalseUnstructured.Object, &nsLabeledFalse)
	assert.NoError(t, err)
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(nsNotLabeledUnstructured.Object, &nsNotLabeled)
	assert.NoError(t, err)
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(nsTestingUnstructured.Object, &nsTesting)
	assert.NoError(t, err)
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(nsLabeledTrueUpdateModeOffUnstructured.Object, &nsLabeledTrueUpdateModeOff)
	assert.NoError(t, err)
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(nsLabeledTrueUpdateModeAutoUnstructured.Object, &nsLabeledTrueUpdateModeAuto)
	assert.NoError(t, err)
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(nsLabeledResourcePolicyUnstructured.Object, &nsLabeledResourcePolicy)
	assert.NoError(t, err)
}

func Test_vpaUpdateModeForNamespace(t *testing.T) {
	setupVPAForTests(t)

	tests := []struct {
		name       string
		ns         *corev1.Namespace
		explicit   bool
		updateMode vpav1.UpdateMode
	}{
		{
			name:       "unlabeled (default)",
			ns:         &nsNotLabeled,
			explicit:   false,
			updateMode: vpav1.UpdateModeOff,
		},
		{
			name:       "labeled: enabled=false",
			ns:         &nsLabeledFalse,
			explicit:   false,
			updateMode: vpav1.UpdateModeOff,
		},
		{
			name:       "labled: enabled=true",
			ns:         &nsLabeledTrue,
			explicit:   false,
			updateMode: vpav1.UpdateModeOff,
		},
		{
			name:       "labled: enabled=true, vpa-update-mode=off",
			ns:         &nsLabeledTrueUpdateModeOff,
			explicit:   true,
			updateMode: vpav1.UpdateModeOff,
		},
		{
			name:       "labled: enabled=true, vpa-update-mode=auto",
			ns:         &nsLabeledTrueUpdateModeAuto,
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
	setupVPAForTests(t)
	rec := GetInstance()

	tests := []struct {
		name       string
		ns         *corev1.Namespace
		updateMode vpav1.UpdateMode
		vpa        *vpav1.VerticalPodAutoscaler
		controller Controller
	}{
		{
			name:       "example-vpa",
			ns:         &nsLabeledTrueUpdateModeAuto,
			updateMode: vpav1.UpdateModeAuto,
			vpa:        nil,
			controller: Controller{
				APIVersion:   "apps/v1",
				Kind:         "Deployment",
				Name:         "test-vpa",
				Unstructured: nil,
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			mode, _ := vpaUpdateModeForResource(test.ns)
			resourcePolicy, _ := vpaResourcePolicyForResource(test.ns)
			minReplicas, _ := vpaMinReplicasForResource(test.ns)
			vpa := rec.getVPAObject(test.vpa, test.ns, test.controller, mode, resourcePolicy, minReplicas)

			// expected ObjectMeta
			assert.Equal(t, "goldilocks-test-vpa", vpa.Name)
			assert.Equal(t, test.ns.Name, vpa.Namespace)
			assert.Equal(t, utils.VPALabels, vpa.Labels)

			// expected .spec.target
			// workload target matches the vpa name minus the prefix goldilocks-
			targetName := strings.Replace(vpa.Name, "goldilocks-", "", -1)
			assert.Equal(t, targetName, vpa.Spec.TargetRef.Name)
			// update mode is correct for the namespace
			assert.Equal(t, test.updateMode, *vpa.Spec.UpdatePolicy.UpdateMode)
		})
	}
}

func Test_createVPA(t *testing.T) {
	setupVPAForTests(t)
	VPAClient := GetInstance().VPAClient

	// First test the dryrun
	rec := GetInstance()
	rec.DryRun = true

	controller := Controller{
		APIVersion:   "apps/v1",
		Kind:         "Deployment",
		Name:         "test-vpa",
		Unstructured: nil,
	}

	updateMode, _ := vpaUpdateModeForResource(&nsTesting)
	resourcePolicy, _ := vpaResourcePolicyForResource(&nsTesting)
	minReplicas, _ := vpaMinReplicasForResource(&nsTesting)
	testVPA := rec.getVPAObject(nil, &nsTesting, controller, updateMode, resourcePolicy, minReplicas)

	err := rec.createVPA(testVPA)
	assert.NoError(t, err)
	_, err = VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsTesting.Name).Get(context.TODO(), "goldilocks-test-vpa", metav1.GetOptions{})
	assert.EqualError(t, err, "verticalpodautoscalers.autoscaling.k8s.io \"goldilocks-test-vpa\" not found")

	// Now actually create and compare
	rec.DryRun = false
	errCreate := rec.createVPA(testVPA)
	newVPA, _ := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsTesting.Name).Get(context.TODO(), "goldilocks-test-vpa", metav1.GetOptions{})
	assert.NoError(t, errCreate)
	assert.EqualValues(t, &testVPA, newVPA)
}

func Test_createVPAWithResourcePolicy(t *testing.T) {
	setupVPAForTests(t)
	VPAClient := GetInstance().VPAClient

	controller := Controller{
		APIVersion:   "apps/v1",
		Kind:         "Deployment",
		Name:         "test-vpa",
		Unstructured: nil,
	}

	// First test the dryrun
	rec := GetInstance()
	rec.DryRun = false

	updateMode, _ := vpaUpdateModeForResource(&nsLabeledResourcePolicy)
	resourcePolicy, _ := vpaResourcePolicyForResource(&nsLabeledResourcePolicy)
	minReplicas, _ := vpaMinReplicasForResource(&nsLabeledResourcePolicy)
	testVPA := rec.getVPAObject(nil, &nsLabeledResourcePolicy, controller, updateMode, resourcePolicy, minReplicas)

	errCreate := rec.createVPA(testVPA)
	newVPA, _ := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsLabeledResourcePolicy.Name).Get(context.TODO(), "goldilocks-test-vpa", metav1.GetOptions{})
	assert.NoError(t, errCreate)
	assert.EqualValues(t, &testVPA, newVPA)
	assert.NotNil(t, newVPA.Spec.ResourcePolicy)
}

func Test_deleteVPA(t *testing.T) {
	setupVPAForTests(t)
	VPAClient := GetInstance().VPAClient

	// First test the dryrun
	rec := GetInstance()
	rec.DryRun = true

	controller := Controller{
		APIVersion:   "apps/v1",
		Kind:         "Deployment",
		Name:         "test-vpa",
		Unstructured: nil,
	}
	updateMode, _ := vpaUpdateModeForResource(&nsTesting)
	resourcePolicy, _ := vpaResourcePolicyForResource(&nsTesting)
	minReplicas, _ := vpaMinReplicasForResource(&nsTesting)
	testVPA := rec.getVPAObject(nil, &nsTesting, controller, updateMode, resourcePolicy, minReplicas)

	_, err := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsTesting.Name).Create(context.TODO(), &testVPA, metav1.CreateOptions{})
	assert.NoError(t, err)

	errDeleteDryRun := rec.deleteVPA(testVPA)
	assert.NoError(t, errDeleteDryRun)
	oldVPA, _ := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsTesting.Name).Get(context.TODO(), "goldilocks-test-vpa", metav1.GetOptions{})
	assert.EqualValues(t, &testVPA, oldVPA)

	// Test actual deletion
	rec.DryRun = false
	errDelete := rec.deleteVPA(testVPA)
	assert.NoError(t, errDelete)
	_, errNotFound := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers("testing").Get(context.TODO(), "goldilocks-test-vpa", metav1.GetOptions{})
	assert.EqualError(t, errNotFound, "verticalpodautoscalers.autoscaling.k8s.io \"goldilocks-test-vpa\" not found")
}

func Test_updateVPA(t *testing.T) {
	setupVPAForTests(t)
	VPAClient := GetInstance().VPAClient

	// First test the dryrun
	rec := GetInstance()
	rec.DryRun = true

	testNS := nsTesting.DeepCopy()
	testNS.Labels["goldilocks.fairwinds.com/vpa-update-mode"] = "off"

	controller := Controller{
		APIVersion:   "apps/v1",
		Kind:         "Deployment",
		Name:         "test-vpa",
		Unstructured: nil,
	}
	updateMode, _ := vpaUpdateModeForResource(testNS)
	resourcePolicy, _ := vpaResourcePolicyForResource(testNS)
	minReplicas, _ := vpaMinReplicasForResource(testNS)
	testVPA := rec.getVPAObject(nil, testNS, controller, updateMode, resourcePolicy, minReplicas)

	_, err := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(testNS.Name).Create(context.TODO(), &testVPA, metav1.CreateOptions{})
	assert.NoError(t, err)

	// dry run
	errUpdateDryRun := rec.updateVPA(testVPA)
	assert.NoError(t, errUpdateDryRun)
	currVPA, _ := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(testNS.Name).Get(context.TODO(), "goldilocks-test-vpa", metav1.GetOptions{})
	assert.EqualValues(t, &testVPA, currVPA)

	// live update
	rec.DryRun = false
	errUpdate := rec.updateVPA(testVPA)
	assert.NoError(t, errUpdate)
	currVPA, _ = VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(testNS.Name).Get(context.TODO(), "goldilocks-test-vpa", metav1.GetOptions{})
	// no change between create and update
	assert.EqualValues(t, &testVPA, currVPA)

	// change the update mode
	testNS.Labels["goldilocks.fairwinds.com/vpa-update-mode"] = "auto"
	updateMode, _ = vpaUpdateModeForResource(testNS)
	resourcePolicy, _ = vpaResourcePolicyForResource(testNS)
	minReplicas, _ = vpaMinReplicasForResource(testNS)
	newVPA := rec.getVPAObject(nil, testNS, controller, updateMode, resourcePolicy, minReplicas)

	errUpdate2 := rec.updateVPA(newVPA)
	assert.NoError(t, errUpdate2)
	currVPA, _ = VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(testNS.Name).Get(context.TODO(), "goldilocks-test-vpa", metav1.GetOptions{})
	// no change between create and update
	assert.NotEqual(t, &testVPA, currVPA)
	// check that the update mode changed
	assert.Equal(t, updateModeAuto, *currVPA.Spec.UpdatePolicy.UpdateMode)
}

func Test_listVPA(t *testing.T) {
	setupVPAForTests(t)
	rec := GetInstance()

	// test namespaces
	testNS1 := nsTesting.DeepCopy()
	testNS1.Name = "ns1"
	testNS1.Namespace = "ns1"
	testNS2 := nsTesting.DeepCopy()
	testNS2.Name = "ns2"
	testNS2.Namespace = "ns2"

	controller1 := Controller{
		APIVersion:   "apps/v1",
		Kind:         "Deployment",
		Name:         "test1",
		Unstructured: nil,
	}
	controller2 := Controller{
		APIVersion:   "apps/v1",
		Kind:         "Deployment",
		Name:         "test2",
		Unstructured: nil,
	}
	controller3 := Controller{
		APIVersion:   "apps/v1",
		Kind:         "Deployment",
		Name:         "test3",
		Unstructured: nil,
	}
	// test vpas
	updateMode1, _ := vpaUpdateModeForResource(testNS1)
	updateMode2, _ := vpaUpdateModeForResource(testNS2)
	resourcePolicy1, _ := vpaResourcePolicyForResource(testNS1)
	resourcePolicy2, _ := vpaResourcePolicyForResource(testNS2)
	minReplicas1, _ := vpaMinReplicasForResource(testNS1)
	minReplicas2, _ := vpaMinReplicasForResource(testNS2)

	vpa1 := rec.getVPAObject(nil, testNS1, controller1, updateMode1, resourcePolicy1, minReplicas1)
	vpa2 := rec.getVPAObject(nil, testNS1, controller2, updateMode1, resourcePolicy1, minReplicas1)
	vpa3 := rec.getVPAObject(nil, testNS2, controller3, updateMode2, resourcePolicy2, minReplicas2)

	// create vpas
	_ = rec.createVPA(vpa1)
	_ = rec.createVPA(vpa2)
	_ = rec.createVPA(vpa3)

	// list ns1
	vpaList1, err := rec.listVPAs("ns1")
	assert.NoError(t, err)
	assert.NotEmpty(t, vpaList1)
	assert.EqualValues(t, vpaList1[0].Name, "goldilocks-test1")
	assert.EqualValues(t, vpaList1[1].Name, "goldilocks-test2")

	// list all
	vpaList2, err := rec.listVPAs("")
	assert.NoError(t, err)
	assert.NotEmpty(t, vpaList2)
	assert.EqualValues(t, vpaList2[0].Name, "goldilocks-test1")
	assert.EqualValues(t, vpaList2[1].Name, "goldilocks-test2")
	assert.EqualValues(t, vpaList2[2].Name, "goldilocks-test3")

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
			namespace: &nsLabeledTrue,
			want:      true,
		},
		{
			name:      "Labeled Incorrectly",
			namespace: &nsLabeledFalse,
			want:      false,
		},
		{
			name:      "Not Labeled",
			namespace: &nsNotLabeled,
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
	setupVPAForTests(t)
	vpaReconciler := GetInstance()

	vpaReconciler.OnByDefault = false
	vpaReconciler.IncludeNamespaces = []string{}
	vpaReconciler.ExcludeNamespaces = []string{}
	got := vpaReconciler.namespaceIsManaged(&nsNotLabeled)
	assert.Equal(t, false, got)

	vpaReconciler.OnByDefault = true
	vpaReconciler.IncludeNamespaces = []string{}
	vpaReconciler.ExcludeNamespaces = []string{}
	got = vpaReconciler.namespaceIsManaged(&nsNotLabeled)
	assert.Equal(t, true, got)

	vpaReconciler.OnByDefault = false
	vpaReconciler.IncludeNamespaces = []string{nsNotLabeled.ObjectMeta.Name}
	vpaReconciler.ExcludeNamespaces = []string{}
	got = vpaReconciler.namespaceIsManaged(&nsNotLabeled)
	assert.Equal(t, true, got)

	vpaReconciler.OnByDefault = true
	vpaReconciler.IncludeNamespaces = []string{}
	vpaReconciler.ExcludeNamespaces = []string{nsNotLabeled.ObjectMeta.Name}
	got = vpaReconciler.namespaceIsManaged(&nsNotLabeled)
	assert.Equal(t, false, got)

	// Labels take precedence over CLI options
	vpaReconciler.OnByDefault = true
	vpaReconciler.IncludeNamespaces = []string{}
	vpaReconciler.ExcludeNamespaces = []string{}
	got = vpaReconciler.namespaceIsManaged(&nsLabeledFalse)
	assert.Equal(t, false, got)

	vpaReconciler.OnByDefault = false
	vpaReconciler.IncludeNamespaces = []string{nsLabeledFalse.ObjectMeta.Name}
	vpaReconciler.ExcludeNamespaces = []string{}
	got = vpaReconciler.namespaceIsManaged(&nsLabeledFalse)
	assert.Equal(t, false, got)

	vpaReconciler.OnByDefault = false
	vpaReconciler.IncludeNamespaces = []string{}
	vpaReconciler.ExcludeNamespaces = []string{nsLabeledTrue.ObjectMeta.Name}
	got = vpaReconciler.namespaceIsManaged(&nsLabeledTrue)
	assert.Equal(t, true, got)
}

func Test_ReconcileNamespaceNoLabels(t *testing.T) {
	setupVPAForTests(t)
	VPAClient := GetInstance().VPAClient
	DynamicClient := GetInstance().DynamicClient.Client

	// Create ns
	nsName := nsLabeledFalse.ObjectMeta.Name
	_, err := DynamicClient.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}).Create(context.TODO(), nsLabeledFalseUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)

	// Create all deployment objects (deployment, replicaset, and pod)
	_, err = DynamicClient.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}).Namespace(nsName).Create(context.TODO(), testDeploymentUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = DynamicClient.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"}).Namespace(nsName).Create(context.TODO(), testDeploymentReplicaSetUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = DynamicClient.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}).Namespace(nsName).Create(context.TODO(), testDeploymentPodUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)

	// False labels should generate 0 vpa objects
	err = GetInstance().ReconcileNamespace(&nsLabeledFalse)
	assert.NoError(t, err)

	vpaList, err := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsName).List(context.TODO(), metav1.ListOptions{})
	assert.Equal(t, 0, len(vpaList.Items))
	assert.NoError(t, err)
	assert.EqualValues(t, vpaList, &vpav1.VerticalPodAutoscalerList{})
}

func Test_ReconcileNamespaceWithLabels(t *testing.T) {
	setupVPAForTests(t)

	// Create ns
	nsName := nsLabeledTrue.ObjectMeta.Name
	_, err := GetInstance().DynamicClient.Client.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}).Create(context.TODO(), nsLabeledTrueUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)

	// Create all deployment objects (deployment, replicaset, and pod)
	_, err = GetInstance().DynamicClient.Client.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}).Namespace(nsName).Create(context.TODO(), testDeploymentUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = GetInstance().DynamicClient.Client.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"}).Namespace(nsName).Create(context.TODO(), testDeploymentReplicaSetUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = GetInstance().DynamicClient.Client.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}).Namespace(nsName).Create(context.TODO(), testDeploymentPodUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)

	// This should create a single VPA
	err = GetInstance().ReconcileNamespace(&nsLabeledTrue)
	assert.NoError(t, err)

	vpaList, err := GetInstance().VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsName).List(context.TODO(), metav1.ListOptions{})
	assert.NoError(t, err)
	if assert.Equal(t, 1, len(vpaList.Items)) {
		assert.Equal(t, "goldilocks-test-deploy", vpaList.Items[0].ObjectMeta.Name)
	}
}

func Test_ReconcileNamespaceDeleteDeployment(t *testing.T) {
	setupVPAForTests(t)
	VPAClient := GetInstance().VPAClient
	DynamicClient := GetInstance().DynamicClient.Client

	// Create ns
	nsName := nsLabeledTrue.ObjectMeta.Name
	_, err := DynamicClient.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}).Create(context.TODO(), nsLabeledTrueUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)

	// Create all deployment objects (deployment, replicaset, and pod)
	_, err = DynamicClient.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}).Namespace(nsName).Create(context.TODO(), testDeploymentUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = DynamicClient.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"}).Namespace(nsName).Create(context.TODO(), testDeploymentReplicaSetUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = DynamicClient.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}).Namespace(nsName).Create(context.TODO(), testDeploymentPodUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	err = GetInstance().ReconcileNamespace(&nsLabeledTrue)
	assert.NoError(t, err)

	// Delete all deployment objects (deployment, replicaset, and pod)
	err = DynamicClient.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}).Namespace(nsName).Delete(context.TODO(), testDeploymentUnstructured.GetName(), metav1.DeleteOptions{})
	assert.NoError(t, err)
	err = DynamicClient.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"}).Namespace(nsName).Delete(context.TODO(), testDeploymentReplicaSetUnstructured.GetName(), metav1.DeleteOptions{})
	assert.NoError(t, err)
	err = DynamicClient.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}).Namespace(nsName).Delete(context.TODO(), testDeploymentPodUnstructured.GetName(), metav1.DeleteOptions{})
	assert.NoError(t, err)

	err = GetInstance().ReconcileNamespace(&nsLabeledTrue)
	assert.NoError(t, err)

	// No VPA objects left after deleted deployment
	vpaList, err := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsName).List(context.TODO(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Empty(t, vpaList.Items)
}

func Test_ReconcileNamespaceRemoveLabel(t *testing.T) {
	setupVPAForTests(t)
	VPAClient := GetInstance().VPAClient
	DynamicClient := GetInstance().DynamicClient.Client

	// Create ns
	nsName := nsLabeledTrue.ObjectMeta.Name
	_, err := DynamicClient.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}).Create(context.TODO(), nsLabeledTrueUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)

	// Create all deployment objects (deployment, replicaset, and pod)
	_, err = DynamicClient.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}).Namespace(nsName).Create(context.TODO(), testDeploymentUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = DynamicClient.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"}).Namespace(nsName).Create(context.TODO(), testDeploymentReplicaSetUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = DynamicClient.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}).Namespace(nsName).Create(context.TODO(), testDeploymentPodUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)

	// Create a non-goldilocks managed VPA
	_, err = VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsName).Create(context.TODO(), &testConflictingVPA, metav1.CreateOptions{})
	assert.NoError(t, err)

	err = GetInstance().ReconcileNamespace(&nsLabeledTrue)
	assert.NoError(t, err)

	// Update the namespace labels to be false and reconcile
	updatedNS := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind": "Namespace",
			"metadata": map[string]interface{}{
				"name": "labeled-true",
				"labels": map[string]interface{}{
					"goldilocks.fairwinds.com/enabled": "false",
				},
			},
		},
	}
	var updatedNSConverted corev1.Namespace
	_, err = DynamicClient.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}).Update(context.TODO(), updatedNS, metav1.UpdateOptions{})
	assert.NoError(t, err)
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(updatedNS.Object, &updatedNSConverted)
	assert.NoError(t, err)
	err = GetInstance().ReconcileNamespace(&updatedNSConverted)
	assert.NoError(t, err)

	// There should be only non-goldilocks managed VPAs left
	vpaList, err := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsName).List(context.TODO(), metav1.ListOptions{})
	assert.NoError(t, err)
	assertVPAListContains(t, vpaList.Items, testConflictingVPA.Name)
}

func Test_ReconcileNamespace_ExcludeDeploymentAnnotation(t *testing.T) {
	setupVPAForTests(t)
	VPAClient := GetInstance().VPAClient
	DynamicClient := GetInstance().DynamicClient.Client

	// Create ns
	nsName := nsLabeledTrue.ObjectMeta.Name
	_, err := DynamicClient.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}).Create(context.TODO(), nsLabeledTrueUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)

	// Create all deployment objects (deployment, replicaset, and pod)
	_, err = DynamicClient.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}).Namespace(nsName).Create(context.TODO(), testDeploymentExcludedUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = DynamicClient.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"}).Namespace(nsName).Create(context.TODO(), testDeploymentReplicaSetUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = DynamicClient.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}).Namespace(nsName).Create(context.TODO(), testDeploymentPodUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	err = GetInstance().ReconcileNamespace(&nsLabeledTrue)
	assert.NoError(t, err)

	// There should be one vpa object with UpdateModeOff
	vpaList, err := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsName).List(context.TODO(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(vpaList.Items))
	assert.EqualValues(t, *vpaList.Items[0].Spec.UpdatePolicy.UpdateMode, vpav1.UpdateModeOff)
}

func Test_ReconcileNamespace_ChangeUpdateMode(t *testing.T) {
	setupVPAForTests(t)
	VPAClient := GetInstance().VPAClient
	DynamicClient := GetInstance().DynamicClient.Client

	// Create ns
	nsName := nsLabeledTrue.ObjectMeta.Name
	_, err := DynamicClient.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}).Create(context.TODO(), nsLabeledTrueUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)

	// Create a deployment in the namespace and reconcile
	_, err = DynamicClient.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}).Namespace(nsName).Create(context.TODO(), testDeploymentUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = DynamicClient.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"}).Namespace(nsName).Create(context.TODO(), testDeploymentReplicaSetUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = DynamicClient.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}).Namespace(nsName).Create(context.TODO(), testDeploymentPodUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	err = GetInstance().ReconcileNamespace(&nsLabeledTrue)
	assert.NoError(t, err)

	// There should be one vpa object with updatemode "off"
	vpaList, err := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsName).List(context.TODO(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(vpaList.Items))
	assert.EqualValues(t, *vpaList.Items[0].Spec.UpdatePolicy.UpdateMode, vpav1.UpdateModeOff)

	_, err = DynamicClient.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}).Update(context.TODO(), nsLabeledTrueUpdateModeAutoUnstructured, metav1.UpdateOptions{})
	assert.NoError(t, err)
	err = GetInstance().ReconcileNamespace(&nsLabeledTrueUpdateModeAuto)
	assert.NoError(t, err)

	// There should be one vpa object with updatemode "auto"
	vpaList1, err := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsName).List(context.TODO(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(vpaList1.Items))
	assert.EqualValues(t, *vpaList1.Items[0].Spec.UpdatePolicy.UpdateMode, vpav1.UpdateModeAuto)
}

func Test_ReconcileNamespaceDaemonset(t *testing.T) {
	setupVPAForTests(t)
	VPAClient := GetInstance().VPAClient
	DynamicClient := GetInstance().DynamicClient.Client

	// Create ns
	nsName := nsLabeledTrue.ObjectMeta.Name
	_, err := DynamicClient.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}).Create(context.TODO(), nsLabeledTrueUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)

	// Create all deployment objects (deployment, replicaset, and pod)
	_, err = DynamicClient.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}).Namespace(nsName).Create(context.TODO(), testDaemonsetUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = DynamicClient.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}).Namespace(nsName).Create(context.TODO(), testDaemonsetPodUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)

	// This should create a single VPA
	err = GetInstance().ReconcileNamespace(&nsLabeledTrue)
	assert.NoError(t, err)

	vpaList, err := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsName).List(context.TODO(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(vpaList.Items))
	assert.Equal(t, "goldilocks-test-ds", vpaList.Items[0].ObjectMeta.Name)
}

func Test_ReconcileNamespaceStatefulSet(t *testing.T) {
	setupVPAForTests(t)
	VPAClient := GetInstance().VPAClient
	DynamicClient := GetInstance().DynamicClient.Client

	// Create ns
	nsName := nsLabeledTrue.ObjectMeta.Name
	_, err := DynamicClient.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}).Create(context.TODO(), nsLabeledTrueUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)

	// Create all deployment objects (deployment, replicaset, and pod)
	_, err = DynamicClient.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}).Namespace(nsName).Create(context.TODO(), testStatefulsetUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = DynamicClient.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}).Namespace(nsName).Create(context.TODO(), testStatefulsetPodUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)

	// This should create a single VPA
	err = GetInstance().ReconcileNamespace(&nsLabeledTrue)
	assert.NoError(t, err)

	vpaList, err := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsName).List(context.TODO(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(vpaList.Items))
	assert.Equal(t, "goldilocks-test-sts", vpaList.Items[0].ObjectMeta.Name)
}

func Test_ReconcileNamespaceConflictingVPA(t *testing.T) {
	setupVPAForTests(t)
	VPAClient := GetInstance().VPAClient
	DynamicClient := GetInstance().DynamicClient.Client

	// Create ns
	nsName := nsLabeledTrue.ObjectMeta.Name
	_, err := DynamicClient.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}).Create(context.TODO(), nsLabeledTrueUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)

	_, err = DynamicClient.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}).Namespace(nsName).Create(context.TODO(), testDeploymentUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsName).Create(context.TODO(), &testConflictingVPA, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = DynamicClient.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}).Namespace(nsName).Create(context.TODO(), testDeploymentPodUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)

	// This should not create any VPA
	err = GetInstance().ReconcileNamespace(&nsLabeledTrue)
	assert.NoError(t, err)

	// should only get testConflictingVPA in this list
	vpaList, err := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsName).List(context.TODO(), metav1.ListOptions{})
	assert.NoError(t, err)
	assertVPAListContains(t, vpaList.Items, testConflictingVPA.GetName())
}

func Test_ReconcileNamespaceNotConflictingVPA(t *testing.T) {
	setupVPAForTests(t)
	VPAClient := GetInstance().VPAClient
	DynamicClient := GetInstance().DynamicClient.Client

	// Create ns
	nsName := nsLabeledTrue.ObjectMeta.Name
	_, err := DynamicClient.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}).Create(context.TODO(), nsLabeledTrueUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)

	testVPAUnrelated := testConflictingVPA.DeepCopy()
	testVPAUnrelated.Spec.TargetRef.Name = "some-other-deployment"

	_, err = DynamicClient.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}).Namespace(nsName).Create(context.TODO(), testDeploymentUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsName).Create(context.TODO(), testVPAUnrelated, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = DynamicClient.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}).Namespace(nsName).Create(context.TODO(), testDeploymentPodUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)

	// This should still create VPA because the existing VPA points to some other deployment
	err = GetInstance().ReconcileNamespace(&nsLabeledTrue)
	assert.NoError(t, err)

	vpaList, err := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsName).List(context.TODO(), metav1.ListOptions{})
	assert.NoError(t, err)
	assertVPAListContains(t, vpaList.Items, testVPAUnrelated.GetName(), "goldilocks-test-deploy")
}

func Test_ReconcileConflictingHPA(t *testing.T) {
	setupVPAForTests(t)
	VPAClient := GetInstance().VPAClient
	DynamicClient := GetInstance().DynamicClient.Client

	// Create ns
	nsName := nsLabeledTrue.ObjectMeta.Name
	_, err := DynamicClient.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}).Create(context.TODO(), nsLabeledTrueUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)

	_, err = DynamicClient.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}).Namespace(nsName).Create(context.TODO(), testDeploymentUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = GetInstance().KubeClient.Client.AutoscalingV2().HorizontalPodAutoscalers(nsName).Create(context.TODO(), &testConflictingHPA, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = DynamicClient.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}).Namespace(nsName).Create(context.TODO(), testDeploymentPodUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)

	// This should not create VPA because the existing HPA conflicts with VPA
	err = GetInstance().ReconcileNamespace(&nsLabeledTrue)
	assert.NoError(t, err)

	vpaList, err := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsName).List(context.TODO(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Empty(t, vpaList.Items)
}

func Test_ReconcileNotConflictingHPA(t *testing.T) {
	setupVPAForTests(t)
	VPAClient := GetInstance().VPAClient
	DynamicClient := GetInstance().DynamicClient.Client

	// Create ns
	nsName := nsLabeledTrue.ObjectMeta.Name
	_, err := DynamicClient.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}).Create(context.TODO(), nsLabeledTrueUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)

	notConflictingHPA := testConflictingHPA.DeepCopy()
	notConflictingHPA.Spec.ScaleTargetRef.Name = "some-other-deployment"

	_, err = DynamicClient.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}).Namespace(nsName).Create(context.TODO(), testDeploymentUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = GetInstance().KubeClient.Client.AutoscalingV2().HorizontalPodAutoscalers(nsName).Create(context.TODO(), notConflictingHPA, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = DynamicClient.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}).Namespace(nsName).Create(context.TODO(), testDeploymentPodUnstructured, metav1.CreateOptions{})
	assert.NoError(t, err)

	// This should not create VPA because the existing HPA conflicts with VPA
	err = GetInstance().ReconcileNamespace(&nsLabeledTrue)
	assert.NoError(t, err)

	vpaList, err := VPAClient.Client.AutoscalingV1().VerticalPodAutoscalers(nsName).List(context.TODO(), metav1.ListOptions{})
	assert.NoError(t, err)
	assertVPAListContains(t, vpaList.Items, "goldilocks-test-deploy")
}

func assertVPAListContains(t *testing.T, vpas []vpav1.VerticalPodAutoscaler, names ...string) {
	t.Helper()

	gotNames := make(map[string]struct{}, len(vpas))
	for _, vpa := range vpas {
		gotNames[vpa.Name] = struct{}{}
	}
	wantNames := make(map[string]struct{}, len(names))
	for _, name := range names {
		wantNames[name] = struct{}{}
	}
	assert.Equal(t, wantNames, gotNames)
}
