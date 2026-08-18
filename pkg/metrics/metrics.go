// Copyright 2019 FairwindsOps Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package metrics exposes Prometheus metrics for the goldilocks controller.
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const namespace = "goldilocks"

var (
	// EventsProcessedTotal counts the Kubernetes watch events the controller has
	// processed, broken down by the resource type and the kind of event.
	EventsProcessedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "controller",
		Name:      "events_processed_total",
		Help:      "Total number of Kubernetes watch events processed by the controller.",
	}, []string{"resource", "event_type"})

	// ProcessErrorsTotal counts events that could not be processed successfully
	// after all retries have been exhausted.
	ProcessErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "controller",
		Name:      "process_errors_total",
		Help:      "Total number of watch events that failed to process after all retries.",
	}, []string{"resource"})
)

// Handler returns an http.Handler that serves Prometheus metrics in the
// standard exposition format.
func Handler() http.Handler {
	return promhttp.Handler()
}

// Healthz replies with a zero byte 200 response, suitable for use as a
// liveness or readiness probe.
func Healthz() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
}
