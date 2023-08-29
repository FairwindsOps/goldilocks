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

package dashboard

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"k8s.io/klog/v2"

	"github.com/fairwindsops/goldilocks/pkg/summary"
)

const (
	kibibyte = 1024
	mebibyte = kibibyte * 1024
	gibibyte = mebibyte * 1024
)

// Limit data loss to only 5% due to rounding error.
const roundingThreshold = 10

// Dashboard replies with the rendered dashboard (on the basePath) for the summarizer
func Dashboard(opts Options) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		costPerCPU := r.URL.Query().Get("costPerCPU")
		costPerGB := r.URL.Query().Get("costPerGB")

		var namespace string
		if val, ok := vars["namespace"]; ok {
			namespace = val
		}

		vpaData, err := getVPAData(opts, namespace, costPerCPU, costPerGB)
		if err != nil {
			klog.Errorf("Error getting vpa data %v", err)
			http.Error(w, "Error getting vpa data", http.StatusInternalServerError)
			return
		}

		tmpl, err := getTemplate("dashboard", opts,
			"container",
			"dashboard",
			"filter",
			"namespace",
			"email",
			"api_token",
			"cost_settings",
		)
		if err != nil {
			klog.Errorf("Error getting template data %v", err)
			http.Error(w, "Error getting template data", http.StatusInternalServerError)
			return
		}

		data := struct {
			VpaData summary.Summary
		}{
			VpaData: vpaData,
		}

		writeTemplate(tmpl, opts, &data, w)
	})
}

// API replies with the JSON data of the VPA summary
func API(opts Options) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		costPerCPU := r.URL.Query().Get("costPerCPU")
		costPerGB := r.URL.Query().Get("costPerGB")

		var namespace string
		if val, ok := vars["namespace"]; ok {
			namespace = val
		}

		vpaData, err := getVPAData(opts, namespace, costPerCPU, costPerGB)
		if err != nil {
			klog.Errorf("Error getting vpa data %v", err)
			http.Error(w, "Error getting vpa data", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(vpaData); err != nil {
			klog.Errorf("Error writing vpa data %v", err)
			http.Error(w, "Error writing vpa data", http.StatusInternalServerError)
			return
		}
	})
}

func getVPAData(opts Options, namespace, costPerCPU, costPerGB string) (summary.Summary, error) {

	filterLabels := make(map[string]string)
	if !opts.ShowAllVPAs {
		filterLabels = opts.VpaLabels
	}

	summarizer := summary.NewSummarizer(
		summary.ForNamespace(namespace),
		summary.ForVPAsWithLabels(filterLabels),
		summary.ExcludeContainers(opts.ExcludedContainers),
	)

	vpaData, err := summarizer.GetSummary()
	if err != nil {
		return summary.Summary{}, err
	}

	if costPerCPU != "" && costPerGB != "" {
		costPerCPUFloat, _ := strconv.ParseFloat(costPerCPU, 64)
		costPerGBFloat, _ := strconv.ParseFloat(costPerGB, 64)

		var containerCost, guaranteedCost, burstableCost float64

		for _, n := range vpaData.Namespaces {
			for _, w := range n.Workloads {
				for k, c := range w.Containers {
					containerCost = calculateContainerCost(costPerCPUFloat, costPerGBFloat, c)
					guaranteedCost, burstableCost = calculateRecommendedCosts(costPerCPUFloat, costPerGBFloat, containerCost, c)
					c.ContainerCost = containerCost
					c.ContainerCostInt = getCostInt(containerCost)
					c.GuaranteedCostInt = getCostInt(guaranteedCost)
					c.BurstableCostInt = getCostInt(burstableCost)
					c.GuaranteedCost = math.Abs(guaranteedCost)
					c.BurstableCost = math.Abs(burstableCost)
					w.Containers[k] = c
				}
			}
		}
	}
	return vpaData, nil
}

func calculateContainerCost(costPerCPUFloat float64, costPerGBFloat float64, c summary.ContainerSummary) float64 {
	var cpuRequests, memRequests, cpuLimits, memLimits float64

	if c.Limits != nil {
		cpuLimits = float64(c.Limits.Cpu().MilliValue())
		memLimits = float64(c.Limits.Memory().Value())
	}
	if c.Requests != nil {
		cpuRequests = float64(c.Requests.Cpu().MilliValue())
		memRequests = float64(c.Requests.Memory().Value())
	}

	cpuCost := costPerCPUFloat * getNonZeroAverage(cpuRequests, cpuLimits) / 1000
	memCost := costPerGBFloat * ConvertToGB(int64(getNonZeroAverage(memRequests, memLimits)))

	return toFixed(cpuCost+memCost, 4)
}

func getNonZeroAverage(req, limit float64) float64 {
	if req == 0.0 {
		return limit
	}
	if limit == 0.0 {
		return req
	}
	return (req + limit) / 2.0
}

func calculateRecommendedCosts(costPerCPUFloat float64, costPerGBFloat float64, containerCost float64, c summary.ContainerSummary) (float64, float64) {
	guaranteedCpuCostRecommended := costPerCPUFloat * float64(c.Target.Cpu().MilliValue()) / 1000
	guaranteedMemCosttRecommended := costPerGBFloat * ConvertToGB(c.Target.Memory().Value())

	burstableCpuCostRecommended := costPerCPUFloat * (float64(c.LowerBound.Cpu().MilliValue() + c.UpperBound.Cpu().MilliValue())) / 2 / 1000
	burstableMemCosttRecommended := costPerGBFloat * (ConvertToGB(c.LowerBound.Memory().Value()) + ConvertToGB(c.UpperBound.Memory().Value())) / 2

	guaranteedCostRecommended := guaranteedCpuCostRecommended + guaranteedMemCosttRecommended
	burstableCostRecommended := burstableCpuCostRecommended + burstableMemCosttRecommended

	return toFixed(guaranteedCostRecommended-containerCost, 4), toFixed(burstableCostRecommended-containerCost, 4)
}

func getCostInt(cost float64) int {
	if cost < 0 {
		return -1
	} else if cost > 0 {
		return 1
	}
	return 0
}

func ConvertToGB(memoryValue int64) float64 {
	absoluteValue := memoryValue
	if absoluteValue < 0 {
		absoluteValue = -absoluteValue
	}
	var roundingBase int64 = 1
	convertedMemoryValue := float64(memoryValue)
	if absoluteValue > gibibyte*roundingThreshold {
		convertedMemoryValue = float64((memoryValue / gibibyte) * roundingBase)
	} else if absoluteValue > mebibyte*roundingThreshold {
		convertedMemoryValue = float64(((memoryValue / mebibyte) * roundingBase)) / 1024
	} else if absoluteValue > kibibyte*roundingThreshold {
		convertedMemoryValue = float64(((memoryValue / kibibyte) * roundingBase)) / (1024 * 1024)
	}
	return convertedMemoryValue
}

func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}
