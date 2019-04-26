package summary

import (
	"github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	autoscalingv1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned/typed/autoscaling.k8s.io/v1beta2"
	"k8s.io/client-go/tools/clientcmd"
)

// Run creates a summary of the vpa info
func Run(namespace string, kubeconfig *string, vpaLabels map[string]string) {
	glog.Info("Gathering info for summary.")
	glog.V(3).Infof("Using Kubeconfig: %s", *kubeconfig)
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		glog.Fatal(err.Error())
	}

	vpaClientSet, err := autoscalingv1beta2.NewForConfig(config)
	if err != nil {
		glog.Fatal(err.Error())
	}

	vpaListOptions := metav1.ListOptions{
		LabelSelector: labels.Set(vpaLabels).String(),
	}

	vpas, err := vpaClientSet.VerticalPodAutoscalers(namespace).List(vpaListOptions)
	if err != nil {
		glog.Fatal(err.Error())
	}
	glog.Info(vpas)
}
