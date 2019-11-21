package handlers

import (
	"net/http"

	"k8s.io/klog"
)

// Health replies with the status messages given for healthy
func Health(healthyMessage string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(healthyMessage))
		if err != nil {
			klog.Errorf("Error writing healthcheck: %v", err)
		}
	})
}

// Healthz replies with a zero byte 200 response
func Healthz() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		return
	})
}
