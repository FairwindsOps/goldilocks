package summary

import (
	"github.com/fairwindsops/goldilocks/pkg/kube"
	"github.com/fairwindsops/goldilocks/pkg/utils"
	"k8s.io/apimachinery/pkg/util/sets"
)

// Functional options
type Option func(*Options)

// internal options for getting and caching the Summarizer's VPAs
type Options struct {
	kubeClient         *kube.ClientInstance
	vpaClient          *kube.VPAClientInstance
	namespace          string
	vpaLabels          map[string]string
	excludedContainers sets.String
}

// default options for a Summarizer
func defaultOptions() *Options {
	return &Options{
		kubeClient:         kube.GetInstance(),
		vpaClient:          kube.GetVPAInstance(),
		namespace:          namespaceAllNamespaces,
		vpaLabels:          utils.VPALabels,
		excludedContainers: sets.NewString(),
	}
}

// Option for limiting the summary to a single namespace
func ForNamespace(namespace string) Option {
	return func(opts *Options) {
		opts.namespace = namespace
	}
}

// Option for excluding containers in the summary
func ExcludeContainers(excludedContainers sets.String) Option {
	return func(opts *Options) {
		opts.excludedContainers = excludedContainers
	}
}

// Option for limiting the summary to certain VPAs matching the labels
func ForVPAsWithLabels(vpaLabels map[string]string) Option {
	return func(opts *Options) {
		opts.vpaLabels = vpaLabels
	}
}
