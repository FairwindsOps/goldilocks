package summary

import (
	"github.com/fairwindsops/goldilocks/pkg/kube"
	"github.com/fairwindsops/goldilocks/pkg/utils"
	"k8s.io/apimachinery/pkg/util/sets"
)

type Option func(*options)

// options for getting and caching the Summarizer's VPAs
type options struct {
	kubeClient            *kube.ClientInstance
	vpaClient             *kube.VPAClientInstance
	dynamicClient         *kube.DynamicClientInstance
	controllerUtilsClient *kube.ControllerUtilsClientInstance
	namespace             string
	vpaLabels             map[string]string
	excludedContainers    sets.Set[string]
	useMemoryBinarySI     bool
}

// defaultOptions for a Summarizer
func defaultOptions() *options {
	return &options{
		kubeClient:            kube.GetInstance(),
		vpaClient:             kube.GetVPAInstance(),
		dynamicClient:         kube.GetDynamicInstance(),
		controllerUtilsClient: kube.GetControllerUtilsInstance(),
		namespace:             namespaceAllNamespaces,
		vpaLabels:             utils.VPALabels,
		excludedContainers:    sets.Set[string]{},
	}
}

// ForNamespace is an Option for limiting the summary to a single namespace
func ForNamespace(namespace string) Option {
	return func(opts *options) {
		opts.namespace = namespace
	}
}

// ExcludeContainers is an Option for excluding containers in the summary
func ExcludeContainers(excludedContainers sets.Set[string]) Option {
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

// UseMemoryBinarySI is an Option for using binary SI units for memory (base 2) instead of decimal SI units (base 10)
func UseMemoryBinarySI(useMemoryBinarySI bool) Option {
	return func(opts *options) {
		opts.useMemoryBinarySI = useMemoryBinarySI
	}
}
