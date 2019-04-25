package vpa

import (
	"time"

	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	autoscaling "k8s.io/api/autoscaling/v1"

	v1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2"
	autoscalingv1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned/typed/autoscaling.k8s.io/v1beta2"
)

// Create makes a vpa for every deployment in the namespace
func Create(namespace string, kubeconfig *string) {
	glog.V(3).Infof("Using Kubeconfig: %s", *kubeconfig)
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		glog.Fatal(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatal(err.Error())
	}

	vpaClientSet, err := autoscalingv1beta2.NewForConfig(config)
	if err != nil {
		glog.Fatal(err.Error())
	}

	// List all the deployments
	for {
		deployments, err := clientset.ExtensionsV1beta1().Deployments(namespace).List(metav1.ListOptions{})
		if err != nil {
			glog.Fatal(err.Error())
		}
		glog.Infof("There are %d deployments in the namespace\n", len(deployments.Items))
		var deploymentName string
		for _, deployment := range deployments.Items {
			deploymentName = deployment.ObjectMeta.Name
			glog.Infof("%v\n", deployment.ObjectMeta.Name)

			updateMode := v1beta2.UpdateModeOff
			vpa := &v1beta2.VerticalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name: deploymentName,
				},
				Spec: v1beta2.VerticalPodAutoscalerSpec{
					TargetRef: &autoscaling.CrossVersionObjectReference{
						APIVersion: "extensions/v1beta1",
						Kind:       "Deployment",
						Name:       deploymentName,
					},
					UpdatePolicy: &v1beta2.PodUpdatePolicy{
						UpdateMode: &updateMode,
					},
				},
			}

			glog.Infof("Creating vpa: %s", deploymentName)
			glog.V(9).Infof("%v", vpa)

			_, err := vpaClientSet.VerticalPodAutoscalers(namespace).Create(vpa)
			if err != nil {
				glog.Error(err)
			}
		}

		time.Sleep(10 * time.Second)
	}
}
