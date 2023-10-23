package handler

import (
	"github.com/davecgh/go-spew/spew"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/klog/v2"

	"github.com/fairwindsops/goldilocks/pkg/kube"
	"github.com/fairwindsops/goldilocks/pkg/utils"
	goldilocksvpa "github.com/fairwindsops/goldilocks/pkg/vpa"
)

func OnVPAChanged(vpa *vpav1.VerticalPodAutoscaler, event utils.Event) {
	kubeClient := kube.GetInstance()
	namespace, err := kube.GetNamespace(kubeClient, event.Namespace)
	if err != nil {
		klog.Error("handler got error retrieving namespace object. breaking.")
		klog.V(5).Info("dumping out event struct")
		klog.V(5).Info(spew.Sdump(event))
		return
	}

	klog.V(3).Infof("VPA %s/%s updated. Reconciling namespace.", vpa.Namespace, vpa.Name)
	err = goldilocksvpa.GetInstance().ReconcileNamespace(namespace)
	if err != nil {
		klog.Errorf("Error reconciling: %v", err)
	}
}
