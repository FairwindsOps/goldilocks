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

package utils

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
)

var (
	// LabelBase is the string that will be used for labels on namespaces
	LabelOrAnnotationBase = "goldilocks.fairwinds.com"
	// VpaEnabledLabel is the label used to indicate that Goldilocks is enabled.
	VpaEnabledLabel = LabelOrAnnotationBase + "/" + "enabled"
	// VpaUpdateModeKey is the label used to indicate the vpa update mode.
	VpaUpdateModeKey = LabelOrAnnotationBase + "/" + "vpa-update-mode"
	// VpaMinReplicas is the annotation to use to define minimum replicas for eviction of a VPA
	VpaMinReplicasAnnotation = LabelOrAnnotationBase + "/" + "vpa-min-replicas"
	// DeploymentExcludeContainersAnnotation is the label used to exclude container names from being reported.
	WorkloadExcludeContainersAnnotation = LabelOrAnnotationBase + "/" + "exclude-containers"
	// VpaResourcePolicyAnnotation is the annotation use to define the json configuration of PodResourcePolicy section of a vpa
	VpaResourcePolicyAnnotation = LabelOrAnnotationBase + "/" + "vpa-resource-policy"
)

// VPALabels is a set of default labels that get placed on every VPA.
var VPALabels = map[string]string{
	"creator": "Fairwinds",
	"source":  "goldilocks",
}

// An Event represents an update of a Kubernetes object and contains metadata about the update.
type Event struct {
	Key          string // A key identifying the object.  This is in the format <object-type>/<object-name>
	EventType    string // The type of event - update, delete, or create
	Namespace    string // The namespace of the event's object
	ResourceType string // The type of resource that was updated.
}

// UniqueString returns a unique string from a slice.
func UniqueString(stringSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range stringSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

// Difference returns the difference betwee two string slices.
func Difference(a, b []string) (diff []string) {
	m := make(map[string]bool)

	for _, item := range b {
		m[item] = true
	}

	for _, item := range a {
		if _, ok := m[item]; !ok {
			diff = append(diff, item)
		}
	}
	return
}

// FormatResourceList scales the units of a ResourceList so that they are
// human readable
func FormatResourceList(rl v1.ResourceList) v1.ResourceList {
	memoryScales := []resource.Scale{
		resource.Kilo,
		resource.Mega,
		resource.Giga,
		resource.Tera,
	}
	if mem, exists := rl[v1.ResourceMemory]; exists {
		i := 0
		maxAllowableStringLen := 5
		for len(mem.String()) > maxAllowableStringLen && i < len(memoryScales)-1 {
			mem.RoundUp(memoryScales[i])
			i++
		}
		rl[v1.ResourceMemory] = mem
	}
	return rl
}

// IsRetryableError determines if an error should trigger a retry with backoff
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for API server errors that indicate etcd issues
	if errors.IsTimeout(err) {
		return true
	}
	if errors.IsServerTimeout(err) {
		return true
	}
	if errors.IsServiceUnavailable(err) {
		return true
	}
	if errors.IsInternalError(err) {
		return true
	}

	// Check for connection errors and watch stream failures
	errorStr := strings.ToLower(err.Error())
	if strings.Contains(errorStr, "context deadline exceeded") ||
		strings.Contains(errorStr, "context canceled") ||
		strings.Contains(errorStr, "connection refused") ||
		strings.Contains(errorStr, "connection reset") ||
		strings.Contains(errorStr, "transport: authentication handshake failed") ||
		strings.Contains(errorStr, "etcd") ||
		strings.Contains(errorStr, "timeout") ||
		strings.Contains(errorStr, "unable to decode an event from the watch stream") ||
		strings.Contains(errorStr, "watch stream") ||
		strings.Contains(errorStr, "too many requests") {
		return true
	}

	// Check for network errors
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout() || netErr.Temporary()
	}

	return false
}

// IsRBACError determines if an error is due to RBAC permissions (not a control plane issue)
func IsRBACError(err error) bool {
	if err == nil {
		return false
	}
	
	// Check for Kubernetes RBAC errors
	if errors.IsForbidden(err) {
		return true
	}
	if errors.IsUnauthorized(err) {
		return true
	}
	
	// Check for common RBAC error message patterns
	errorStr := err.Error()
	return strings.Contains(errorStr, "is forbidden:") ||
		strings.Contains(errorStr, "cannot list resource") ||
		strings.Contains(errorStr, "cannot get resource") ||
		strings.Contains(errorStr, "cannot create resource") ||
		strings.Contains(errorStr, "cannot update resource") ||
		strings.Contains(errorStr, "cannot delete resource") ||
		strings.Contains(errorStr, "User \"system:serviceaccount:") ||
		// Additional patterns from controller-utils library RBAC errors
		strings.Contains(errorStr, "forbidden: User \"system:serviceaccount:")
}

// RetryWithExponentialBackoff retries a function with exponential backoff for etcd-related failures
func RetryWithExponentialBackoff(ctx context.Context, operation func(ctx context.Context) error, operationName string) error {
	backoff := wait.Backoff{
		Duration: 1 * time.Second,   // Initial delay
		Factor:   2.0,               // Exponential factor
		Jitter:   0.1,               // Add jitter to avoid thundering herd
		Steps:    5,                 // Maximum retry attempts
		Cap:      30 * time.Second,  // Maximum delay between retries
	}

	var lastErr error
	retryCount := 0

	err := wait.ExponentialBackoffWithContext(ctx, backoff, func(ctx context.Context) (bool, error) {
		retryCount++
		lastErr = operation(ctx)
		if lastErr == nil {
			if retryCount > 1 {
				klog.Infof("Operation %s succeeded after %d attempts", operationName, retryCount)
			}
			return true, nil // Success, stop retrying
		}

		if !IsRetryableError(lastErr) {
			klog.V(2).Infof("Operation %s failed with non-retryable error: %v", operationName, lastErr)
			return false, lastErr // Non-retryable error, stop immediately
		}

		klog.V(2).Infof("Operation %s failed (attempt %d), will retry: %v", operationName, retryCount, lastErr)
		return false, nil // Retryable error, continue
	})

	if err != nil {
		if err == wait.ErrWaitTimeout {
			return fmt.Errorf("operation %s failed after %d attempts, last error: %v", operationName, retryCount, lastErr)
		}
		return err
	}

	return nil
}

// CreateContextWithTimeout creates a context with appropriate timeout for API operations
func CreateContextWithTimeout(parentCtx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = 30 * time.Second // Default timeout
	}
	return context.WithTimeout(parentCtx, timeout)
}
