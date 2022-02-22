package dashboard

import (
	"github.com/fairwindsops/goldilocks/pkg/utils"
	"k8s.io/apimachinery/pkg/util/sets"
)

// Option is a Functional options
type Option func(*Options)

// Options are options for getting and caching the Summarizer's VPAs
type Options struct {
	port               int
	basePath           string
	vpaLabels          map[string]string
	excludedContainers sets.String
	onByDefault        bool
	showAllVPAs        bool
}

// default options for the dashboard
func defaultOptions() *Options {
	return &Options{
		port:               8080,
		basePath:           "/",
		vpaLabels:          utils.VPALabels,
		excludedContainers: sets.NewString(),
		onByDefault:        false,
		showAllVPAs:        false,
	}
}

// OnPort is an Option for running the dashboard on a different port
func OnPort(port int) Option {
	return func(opts *Options) {
		opts.port = port
	}
}

// ExcludeContainers is an Option for excluding containers in the dashboard summary
func ExcludeContainers(excludedContainers sets.String) Option {
	return func(opts *Options) {
		opts.excludedContainers = excludedContainers
	}
}

// ForVPAsWithLabels Option for limiting the dashboard to certain VPAs matching the labels
func ForVPAsWithLabels(vpaLabels map[string]string) Option {
	return func(opts *Options) {
		opts.vpaLabels = vpaLabels
	}
}

// OnByDefault is an option for listing all namespaces in the dashboard unless explicitly excluded
func OnByDefault(onByDefault bool) Option {
	return func(opts *Options) {
		opts.onByDefault = onByDefault
	}
}

func ShowAllVPAs(showAllVPAs bool) Option {
	return func(opts *Options) {
		opts.showAllVPAs = showAllVPAs
	}
}
