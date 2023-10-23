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
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
)

// Some namespaces that can be used for tests
var nsLabeledTrue corev1.Namespace
var nsLabeledTrueUnstructured = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind": "Namespace",
		"metadata": map[string]interface{}{
			"name": "labeled-true",
			"labels": map[string]interface{}{
				"goldilocks.fairwinds.com/enabled": "true",
			},
		},
	},
}

var nsLabeledFalse corev1.Namespace
var nsLabeledFalseUnstructured = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind": "Namespace",
		"metadata": map[string]interface{}{
			"name": "labeled-false",
			"labels": map[string]interface{}{
				"goldilocks.fairwinds.com/enabled": "false",
			},
		},
	},
}

var nsNotLabeled corev1.Namespace
var nsNotLabeledUnstructured = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind": "Namespace",
		"metadata": map[string]interface{}{
			"name": "not-labeled",
		},
	},
}

var nsTesting corev1.Namespace
var nsTestingUnstructured = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind": "Namespace",
		"metadata": map[string]interface{}{
			"name": "testing",
			"labels": map[string]interface{}{
				"goldilocks.fairwinds.com/enabled": "True",
			},
		},
	},
}

var nsLabeledTrueUpdateModeOff corev1.Namespace
var nsLabeledTrueUpdateModeOffUnstructured = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind": "Namespace",
		"metadata": map[string]interface{}{
			"name": "labeled-true",
			"labels": map[string]interface{}{
				"goldilocks.fairwinds.com/enabled":         "True",
				"goldilocks.fairwinds.com/vpa-update-mode": "off",
			},
		},
	},
}

var nsLabeledTrueUpdateModeAuto corev1.Namespace
var nsLabeledTrueUpdateModeAutoUnstructured = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind": "Namespace",
		"metadata": map[string]interface{}{
			"name": "labeled-true",
			"labels": map[string]interface{}{
				"goldilocks.fairwinds.com/enabled":         "True",
				"goldilocks.fairwinds.com/vpa-update-mode": "auto",
			},
		},
	},
}

var nsLabeledResourcePolicy corev1.Namespace
var nsLabeledResourcePolicyUnstructured = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind": "Namespace",
		"metadata": map[string]interface{}{
			"name": "labeled-true",
			"labels": map[string]interface{}{
				"goldilocks.fairwinds.com/enabled":         "True",
				"goldilocks.fairwinds.com/vpa-update-mode": "auto",
			},
			"annotations": map[string]interface{}{
				"goldilocks.fairwinds.com/vpa-resource-policy": `
                {
					"containerPolicies":
					[ 
						{ 
						  "containerName": "nginx",
					      "minAllowed": { "cpu": "250m", "memory": "100Mi" },
					      "maxAllowed": {	"cpu": "2000m", "memory": "2048Mi" }
					    },
					    {
						  "containerName": "istio-proxy",
						  "mode": "Off"
					    }
				    ]
			    }
			`,
			},
		},
	},
}

var updateModeAuto = vpav1.UpdateModeAuto

var testDeploymentPodUnstructured = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind":       "Pod",
		"apiVersion": "v1",
		"metadata": map[string]interface{}{
			"ownerReferences": []interface{}{
				map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "ReplicaSet",
					"controller": true,
					"name":       "test-deploy-0123456789",
				},
			},
			"name": "test-deploy-0123456789-01234",
		},
		"spec": map[string]interface{}{},
	},
}

var testDeploymentReplicaSetUnstructured = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind":       "ReplicaSet",
		"apiVersion": "apps/v1",
		"metadata": map[string]interface{}{
			"ownerReferences": []interface{}{
				map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"controller": true,
					"name":       "test-deploy",
				},
			},
			"name": "test-deploy-0123456789",
		},
		"spec": map[string]interface{}{},
	},
}

var testDeploymentUnstructured = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind":       "Deployment",
		"apiVersion": "apps/v1",
		"metadata": map[string]interface{}{
			"name": "test-deploy",
		},
		"spec": map[string]interface{}{},
	},
}

var testConflictingVPA = vpav1.VerticalPodAutoscaler{
	ObjectMeta: metav1.ObjectMeta{
		Name: "conflicting-vpa",
	},
	Spec: vpav1.VerticalPodAutoscalerSpec{
		TargetRef: &autoscalingv1.CrossVersionObjectReference{
			APIVersion: testDeploymentUnstructured.GetAPIVersion(),
			Kind:       testDeploymentUnstructured.GroupVersionKind().Kind,
			Name:       testDeploymentUnstructured.GetName(),
		},
	},
}

var testConflictingHPA = autoscalingv2.HorizontalPodAutoscaler{
	ObjectMeta: metav1.ObjectMeta{
		Name: "conflicting-hpa",
	},
	Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
		ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
			APIVersion: testDeploymentUnstructured.GetAPIVersion(),
			Kind:       testDeploymentUnstructured.GroupVersionKind().Kind,
			Name:       testDeploymentUnstructured.GetName(),
		},
	},
}

var testDeploymentExcludedUnstructured = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind":       "Deployment",
		"apiVersion": "apps/v1",
		"metadata": map[string]interface{}{
			"name": "test-deploy",
			"annotations": map[string]interface{}{
				"goldilocks.fairwinds.com/vpa-update-mode": "off",
			},
		},
		"spec": map[string]interface{}{},
	},
}

var testDaemonsetUnstructured = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind":       "DaemonSet",
		"apiVersion": "apps/v1",
		"metadata": map[string]interface{}{
			"name": "test-ds",
		},
		"spec": map[string]interface{}{},
	},
}

var testDaemonsetPodUnstructured = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind":       "Pod",
		"apiVersion": "v1",
		"metadata": map[string]interface{}{
			"ownerReferences": []interface{}{
				map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "DaemonSet",
					"controller": true,
					"name":       "test-ds",
				},
			},
			"name": "test-ds-01234",
			"spec": map[string]interface{}{},
		},
	},
}

var testStatefulsetUnstructured = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind":       "StatefulSet",
		"apiVersion": "apps/v1",
		"metadata": map[string]interface{}{
			"name": "test-sts",
		},
		"spec": map[string]interface{}{},
	},
}

var testStatefulsetPodUnstructured = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind":       "Pod",
		"apiVersion": "v1",
		"metadata": map[string]interface{}{
			"ownerReferences": []interface{}{
				map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "StatefulSet",
					"controller": true,
					"name":       "test-sts",
				},
			},
			"name": "test-sts-01234",
			"spec": map[string]interface{}{},
		},
	},
}
