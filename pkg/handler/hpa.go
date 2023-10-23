package handler

import (
	"github.com/davecgh/go-spew/spew"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/klog/v2"

	"github.com/fairwindsops/goldilocks/pkg/kube"
	"github.com/fairwindsops/goldilocks/pkg/utils"
	"github.com/fairwindsops/goldilocks/pkg/vpa"
)

func OnHPAChanged(hpa *autoscalingv2.HorizontalPodAutoscaler, event utils.Event) {
	kubeClient := kube.GetInstance()
	namespace, err := kube.GetNamespace(kubeClient, event.Namespace)
	if err != nil {
		klog.Error("handler got error retrieving namespace object. breaking.")
		klog.V(5).Info("dumping out event struct")
		klog.V(5).Info(spew.Sdump(event))
		return
	}

	klog.V(3).Infof("HPA %s/%s updated. Reconciling namespace.", hpa.Namespace, hpa.Name)
	err = vpa.GetInstance().ReconcileNamespace(namespace)
	if err != nil {
		klog.Errorf("Error reconciling: %v", err)
	}
}
