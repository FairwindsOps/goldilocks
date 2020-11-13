package dashboard

import (
	"github.com/fairwindsops/goldilocks/pkg/utils"
	"k8s.io/apimachinery/pkg/util/sets"
)

// Functional options
type Option func(*Options)

// internal options for getting and caching the Summarizer's VPAs
type Options struct {
	port               int
	basePath           string
	vpaLabels          map[string]string
	excludedContainers sets.String
	kubeconfigPath     string
}

// default options for the dashboard
func defaultOptions() *Options {
	return &Options{
		port:               8080,
		basePath:           "/",
		vpaLabels:          utils.VPALabels,
		excludedContainers: sets.NewString(),
	}
}

// Option for running the dashboard on a different port
func OnPort(port int) Option {
	return func(opts *Options) {
		opts.port = port
	}
}

// Option for running the dashboard on a different base path
func WithBasePath(basePath string) Option {
	return func(opts *Options) {
		opts.basePath = basePath
	}
}

// Option for excluding containers in the dashboard summary
func ExcludeContainers(excludedContainers sets.String) Option {
	return func(opts *Options) {
		opts.excludedContainers = excludedContainers
	}
}

// Option for limiting the dashboard to certain VPAs matching the labels
func ForVPAsWithLabels(vpaLabels map[string]string) Option {
	return func(opts *Options) {
		opts.vpaLabels = vpaLabels
	}
}

// Option for setting kubeconfgi
func WithKubeconfig(kubeconfigPath string) Option {
	return func(opts *Options) {
		opts.kubeconfigPath = kubeconfigPath
	}
}
