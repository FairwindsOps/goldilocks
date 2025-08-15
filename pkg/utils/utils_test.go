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
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestUniqueString(t *testing.T) {
	for _, tc := range testUniqueStringCases {
		res := UniqueString(tc.testData)
		assert.Equal(t, res, tc.expected)
	}
}

var testUniqueStringCases = []struct {
	description string
	testData    []string
	expected    []string
}{
	{
		description: "three duplicates, one output",
		testData:    []string{"test", "test", "test"},
		expected:    []string{"test"},
	},
	{
		description: "no duplicates",
		testData:    []string{"one", "two", "three"},
		expected:    []string{"one", "two", "three"},
	},
}

func TestDifference(t *testing.T) {
	for _, tc := range testDifferenceCases {
		res := Difference(tc.testData1, tc.testData2)
		assert.Equal(t, res, tc.expected)
	}
}

var empty []string

var testDifferenceCases = []struct {
	description string
	testData1   []string
	testData2   []string
	expected    []string
}{
	{
		description: "empty case",
		testData1:   []string{"a", "b", "c"},
		testData2:   []string{"a", "b", "c"},
		expected:    empty,
	},
	{
		description: "extra item on right",
		testData1:   []string{"a", "b"},
		testData2:   []string{"a", "b", "c"},
		expected:    empty,
	},
	{
		description: "extra item on left",
		testData1:   []string{"a", "b", "c"},
		testData2:   []string{"a", "b"},
		expected:    []string{"c"},
	},
}

func TestFormatResourceList(t *testing.T) {
	for _, tc := range testFormatResourceCases {
		res := FormatResourceList(tc.testData)
		resource := res[tc.resourceType]
		got := resource.String()
		assert.Equal(t, tc.expected, got)
	}
}

var testFormatResourceCases = []struct {
	description  string
	testData     v1.ResourceList
	resourceType v1.ResourceName
	expected     string
}{
	{
		description: "Unmodified cpu",
		testData: v1.ResourceList{
			"cpu": resource.MustParse("1"),
		},
		resourceType: "cpu",
		expected:     "1",
	},
	{
		description: "Unmodified memory",
		testData: v1.ResourceList{
			"memory": resource.MustParse("1Mi"),
		},
		resourceType: "memory",
		expected:     "1Mi",
	},
	{
		description: "Memory in too large of units",
		testData: v1.ResourceList{
			"memory": resource.MustParse("123456k"),
		},
		resourceType: "memory",
		expected:     "124M",
	},
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "context deadline exceeded",
			err:      errors.New("context deadline exceeded"),
			expected: true,
		},
		{
			name:     "context canceled",
			err:      errors.New("context canceled"),
			expected: true,
		},
		{
			name:     "etcd connection error",
			err:      errors.New("transport: authentication handshake failed: context canceled"),
			expected: true,
		},
		{
			name:     "timeout error",
			err:      errors.New("operation timeout"),
			expected: true,
		},
		{
			name:     "etcd error",
			err:      errors.New("etcd server unavailable"),
			expected: true,
		},
		{
			name:     "connection refused",
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "connection reset",
			err:      errors.New("connection reset by peer"),
			expected: true,
		},
		{
			name:     "kubernetes timeout error",
			err:      kerrors.NewTimeoutError("operation timed out", 30),
			expected: true,
		},
		{
			name:     "kubernetes server timeout error",
			err:      kerrors.NewServerTimeout(schema.GroupResource{Group: "", Resource: "pods"}, "list", 30),
			expected: true,
		},
		{
			name:     "kubernetes service unavailable error",
			err:      kerrors.NewServiceUnavailable("service temporarily unavailable"),
			expected: true,
		},
		{
			name:     "kubernetes internal error",
			err:      kerrors.NewInternalError(errors.New("internal server error")),
			expected: true,
		},
		{
			name:     "non-retryable error",
			err:      kerrors.NewBadRequest("invalid request"),
			expected: false,
		},
		{
			name:     "watch stream decode error",
			err:      errors.New("unable to decode an event from the watch stream: context canceled"),
			expected: true,
		},
		{
			name:     "watch stream error",
			err:      errors.New("watch stream connection failed"),
			expected: true,
		},
		{
			name:     "too many requests error",
			err:      errors.New("too many requests, please try again later"),
			expected: true,
		},
		{
			name:     "generic error",
			err:      errors.New("some other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("IsRetryableError() = %v, want %v for error: %v", result, tt.expected, tt.err)
			}
		})
	}
}

func TestRetryWithExponentialBackoff(t *testing.T) {
	t.Run("succeeds on first attempt", func(t *testing.T) {
		callCount := 0
		operation := func(ctx context.Context) error {
			callCount++
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := RetryWithExponentialBackoff(ctx, operation, "test operation")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if callCount != 1 {
			t.Errorf("Expected 1 call, got %d", callCount)
		}
	})

	t.Run("succeeds after retries", func(t *testing.T) {
		callCount := 0
		operation := func(ctx context.Context) error {
			callCount++
			if callCount < 3 {
				return errors.New("context deadline exceeded") // retryable error
			}
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := RetryWithExponentialBackoff(ctx, operation, "test operation")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if callCount != 3 {
			t.Errorf("Expected 3 calls, got %d", callCount)
		}
	})

	t.Run("fails with non-retryable error", func(t *testing.T) {
		callCount := 0
		expectedErr := kerrors.NewBadRequest("invalid request")
		operation := func(ctx context.Context) error {
			callCount++
			return expectedErr
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := RetryWithExponentialBackoff(ctx, operation, "test operation")
		if err != expectedErr {
			t.Errorf("Expected %v, got %v", expectedErr, err)
		}
		if callCount != 1 {
			t.Errorf("Expected 1 call, got %d", callCount)
		}
	})

	t.Run("exhausts retries", func(t *testing.T) {
		callCount := 0
		operation := func(ctx context.Context) error {
			callCount++
			return errors.New("context deadline exceeded") // retryable error
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := RetryWithExponentialBackoff(ctx, operation, "test operation")
		if err == nil {
			t.Error("Expected error, got nil")
		}
		// The exact number of retries depends on the backoff implementation
		// We expect at least 3 attempts and at most 5
		if callCount < 3 || callCount > 5 {
			t.Errorf("Expected 3-5 calls, got %d", callCount)
		}
	})
}

func TestIsRBACError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "forbidden error",
			err:      kerrors.NewForbidden(schema.GroupResource{Group: "", Resource: "pods"}, "test", errors.New("forbidden")),
			expected: true,
		},
		{
			name:     "unauthorized error",
			err:      kerrors.NewUnauthorized("unauthorized"),
			expected: true,
		},
		{
			name:     "forbidden message",
			err:      errors.New("clustertunnels.networking.cfargotunnel.com is forbidden: User \"system:serviceaccount:goldilocks:goldilocks-controller\" cannot list resource"),
			expected: true,
		},
		{
			name:     "controller-utils RBAC error",
			err:      errors.New("forbidden: User \"system:serviceaccount:goldilocks:goldilocks-controller\" cannot list resource"),
			expected: true,
		},
		{
			name:     "cannot list resource",
			err:      errors.New("cannot list resource \"clustertunnels\" in API group"),
			expected: true,
		},
		{
			name:     "serviceaccount error",
			err:      errors.New("User \"system:serviceaccount:goldilocks:goldilocks-controller\" cannot access"),
			expected: true,
		},
		{
			name:     "non-RBAC error",
			err:      errors.New("connection refused"),
			expected: false,
		},
		{
			name:     "timeout error",
			err:      errors.New("context deadline exceeded"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRBACError(tt.err)
			if result != tt.expected {
				t.Errorf("IsRBACError() = %v, want %v for error: %v", result, tt.expected, tt.err)
			}
		})
	}
}

func TestCreateContextWithTimeout(t *testing.T) {
	t.Run("creates context with specified timeout", func(t *testing.T) {
		parentCtx := context.Background()
		timeout := 5 * time.Second

		ctx, cancel := CreateContextWithTimeout(parentCtx, timeout)
		defer cancel()

		deadline, ok := ctx.Deadline()
		if !ok {
			t.Error("Expected context to have a deadline")
		}

		expectedDeadline := time.Now().Add(timeout)
		if deadline.Before(expectedDeadline.Add(-time.Second)) || deadline.After(expectedDeadline.Add(time.Second)) {
			t.Errorf("Deadline %v is not close to expected %v", deadline, expectedDeadline)
		}
	})

	t.Run("uses default timeout for zero duration", func(t *testing.T) {
		parentCtx := context.Background()
		timeout := time.Duration(0)

		ctx, cancel := CreateContextWithTimeout(parentCtx, timeout)
		defer cancel()

		deadline, ok := ctx.Deadline()
		if !ok {
			t.Error("Expected context to have a deadline")
		}

		expectedDeadline := time.Now().Add(30 * time.Second) // default timeout
		if deadline.Before(expectedDeadline.Add(-time.Second)) || deadline.After(expectedDeadline.Add(time.Second)) {
			t.Errorf("Deadline %v is not close to expected default %v", deadline, expectedDeadline)
		}
	})
}
