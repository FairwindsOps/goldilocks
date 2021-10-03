package summary

import (
	"github.com/fairwindsops/goldilocks/pkg/kube"
	"github.com/fairwindsops/goldilocks/pkg/utils"
	"k8s.io/apimachinery/pkg/util/sets"
)

type Option func(*options)

// options for getting and caching the Summarizer's VPAs
type options struct {
	kubeClient         *kube.ClientInstance
	vpaClient          *kube.VPAClientInstance
	namespace          string
	vpaLabels          map[string]string
	excludedContainers sets.String
	filter             string
}

// defaultOptions for a Summarizer
func defaultOptions() *options {
	return &options{
		kubeClient:         kube.GetInstance(),
		vpaClient:          kube.GetVPAInstance(),
		namespace:          namespaceAllNamespaces,
		vpaLabels:          utils.VPALabels,
		excludedContainers: sets.NewString(),
		filter:             "all",
	}
}

// ForNamespace is an Option for limiting the summary to a single namespace
func ForNamespace(namespace string) Option {
	return func(opts *options) {
		opts.namespace = namespace
	}
}

// ExcludeContainers is an Option for excluding containers in the summary
func ExcludeContainers(excludedContainers sets.String) Option {
	return func(opts *options) {
		opts.excludedContainers = excludedContainers
	}
}

// ForVPAsWithLabels is an Option for limiting the summary to certain VPAs matching the labels
func ForVPAsWithLabels(vpaLabels map[string]string) Option {
	return func(opts *options) {
		opts.vpaLabels = vpaLabels
	}
}

// WithFilter is an Option for limiting the summary to VPAs with unmet recommendations
func WithFilter(filter string) Option {
	return func(opts *options) {
		if filter == "" {
			opts.filter = "all"
		} else {
			opts.filter = filter
		}
	}
}
