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

package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestHandler_ExposesRegisteredMetrics(t *testing.T) {
	EventsProcessedTotal.WithLabelValues("pod", "create").Inc()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	Handler().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "goldilocks_controller_events_processed_total")
	assert.Contains(t, body, `resource="pod"`)
	assert.Contains(t, body, `event_type="create"`)
}

func TestHealthz(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	Healthz().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Body.String())
}

func TestProcessErrorsTotal_Increments(t *testing.T) {
	before := testutil.ToFloat64(ProcessErrorsTotal.WithLabelValues("namespace"))
	ProcessErrorsTotal.WithLabelValues("namespace").Inc()
	after := testutil.ToFloat64(ProcessErrorsTotal.WithLabelValues("namespace"))

	assert.Equal(t, before+1, after)
}
