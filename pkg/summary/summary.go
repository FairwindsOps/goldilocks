package summary

import (
	"encoding/json"
	"fmt"

	"github.com/golang/glog"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	autoscalingv1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned/typed/autoscaling.k8s.io/v1beta2"
	"k8s.io/client-go/tools/clientcmd"
)

type containerSummary struct {
	LowerBound    v1.ResourceList `json:"lowerBound"`
	UpperBound    v1.ResourceList `json:"upperBound"`
	ContainerName string          `json:"containerName"`
}

type deploymentSummary struct {
	Containers     []containerSummary `json:"containers"`
	DeploymentName string             `json:"deploymentName"`
}

// Summary struct is for storing a summary of recommendation data.
type Summary struct {
	Deployments []deploymentSummary `json:"deployments"`
}

// Run creates a summary of the vpa info
func Run(namespace string, kubeconfig *string, vpaLabels map[string]string) {
	glog.V(3).Infof("Gathering info for summary from namespace: %s", namespace)
	glog.V(3).Infof("Using Kubeconfig: %s", *kubeconfig)
	glog.V(3).Infof("Looking for vpa with labels: %v", vpaLabels)
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
	glog.V(10).Infof("Found vpas: %v", vpas)

	if len(vpas.Items) <= 0 {
		glog.Fatalf("No vpas were found in the %s namespace.", namespace)
	}
	var summary Summary
	for _, vpa := range vpas.Items {
		glog.V(8).Infof("Analyzing vpa: %v", vpa.ObjectMeta.Name)
		glog.V(10).Info(vpa.Status.Recommendation.ContainerRecommendations)

		var deployment deploymentSummary
		deployment.DeploymentName = vpa.ObjectMeta.Name
		if len(vpa.Status.Recommendation.ContainerRecommendations) <= 0 {
			glog.V(2).Infof("No recommendations found in the %v vpa.", deployment.DeploymentName)
			break
		}
		for _, containerRecommendation := range vpa.Status.Recommendation.ContainerRecommendations {
			glog.V(7).Infof("Lower Bound: %v", containerRecommendation.LowerBound)
			glog.V(7).Infof("Upper Bound: %v", containerRecommendation.UpperBound)
			container := containerSummary{
				ContainerName: containerRecommendation.ContainerName,
				UpperBound:    containerRecommendation.UpperBound,
				LowerBound:    containerRecommendation.LowerBound,
			}
			deployment.Containers = append(deployment.Containers, container)
		}
		summary.Deployments = append(summary.Deployments, deployment)
	}
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		glog.Fatalf("Error marshalling JSON: %v", err)
	}
	fmt.Println(string(summaryJSON))
}
