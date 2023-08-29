package dashboard

import (
	"github.com/fairwindsops/goldilocks/pkg/utils"
	"k8s.io/apimachinery/pkg/util/sets"
)

// Option is a Functional options
type Option func(*Options)

// Options are options for getting and caching the Summarizer's VPAs
type Options struct {
	Port               int
	BasePath           string
	VpaLabels          map[string]string
	ExcludedContainers sets.Set[string]
	OnByDefault        bool
	ShowAllVPAs        bool
	InsightsHost       string
	EnableCost         bool
}

// default options for the dashboard
func defaultOptions() *Options {
	return &Options{
		Port:               8080,
		BasePath:           "/",
		VpaLabels:          utils.VPALabels,
		ExcludedContainers: sets.Set[string]{},
		OnByDefault:        false,
		ShowAllVPAs:        false,
		EnableCost:         true,
	}
}

// OnPort is an Option for running the dashboard on a different port
func OnPort(port int) Option {
	return func(opts *Options) {
		opts.Port = port
	}
}

// ExcludeContainers is an Option for excluding containers in the dashboard summary
func ExcludeContainers(excludedContainers sets.Set[string]) Option {
	return func(opts *Options) {
		opts.ExcludedContainers = excludedContainers
	}
}

// ForVPAsWithLabels Option for limiting the dashboard to certain VPAs matching the labels
func ForVPAsWithLabels(vpaLabels map[string]string) Option {
	return func(opts *Options) {
		opts.VpaLabels = vpaLabels
	}
}

// OnByDefault is an option for listing all namespaces in the dashboard unless explicitly excluded
func OnByDefault(onByDefault bool) Option {
	return func(opts *Options) {
		opts.OnByDefault = onByDefault
	}
}

func ShowAllVPAs(showAllVPAs bool) Option {
	return func(opts *Options) {
		opts.ShowAllVPAs = showAllVPAs
	}
}

func BasePath(basePath string) Option {
	return func(opts *Options) {
		opts.BasePath = basePath
	}
}

func InsightsHost(insightsHost string) Option {
	return func(opts *Options) {
		opts.InsightsHost = insightsHost
	}
}

func EnableCost(enableCost bool) Option {
	return func(opts *Options) {
		opts.EnableCost = enableCost
	}
}
