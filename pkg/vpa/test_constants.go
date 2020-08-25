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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
)

// Some namespaces that can be used for tests
var nsLabeledTrue = &corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: "labeled-true",
		Labels: map[string]string{
			"goldilocks.fairwinds.com/enabled": "True",
		},
	},
}

var nsLabeledFalse = &corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: "labeled-false",
		Labels: map[string]string{
			"goldilocks.fairwinds.com/enabled": "false",
		},
	},
}

var nsNotLabeled = &corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: "not-labeled",
	},
}

var nsTesting = &corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: "testing",
		Labels: map[string]string{
			"goldilocks.fairwinds.com/enabled": "True",
		},
	},
}

var nsLabeledTrueUpdateModeOff = &corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: "labeled-true",
		Labels: map[string]string{
			"goldilocks.fairwinds.com/enabled":         "True",
			"goldilocks.fairwinds.com/vpa-update-mode": "off",
		},
	},
}

var nsLabeledTrueUpdateModeAuto = &corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: "labeled-true",
		Labels: map[string]string{
			"goldilocks.fairwinds.com/enabled":         "True",
			"goldilocks.fairwinds.com/vpa-update-mode": "auto",
		},
	},
}

var updateModeAuto = vpav1.UpdateModeAuto

// A deployment object that can be used for testing
var testDeployment = &appsv1.Deployment{
	ObjectMeta: metav1.ObjectMeta{
		Name: "test-deploy",
	},
}

var testDeploymentExcluded = &appsv1.Deployment{
	ObjectMeta: metav1.ObjectMeta{
		Name: "test-deploy",
		Annotations: map[string]string{
			"goldilocks.fairwinds.com/vpa-update-mode": "off",
		},
	},
}
