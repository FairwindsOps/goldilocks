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

		filterLabels := make(map[string]string)
		if !opts.showAllVPAs {
			filterLabels = opts.vpaLabels
		}

		// TODO [hkatz] add caching or refresh button support
		summarizer := summary.NewSummarizer(
			summary.ForNamespace(namespace),
			summary.ForVPAsWithLabels(filterLabels),
			summary.ExcludeContainers(opts.excludedContainers),
		)

		vpaData, err := summarizer.GetSummary()
		if err != nil {
			klog.Errorf("Error getting vpaData: %v", err)
			http.Error(w, "Error running summary.", http.StatusInternalServerError)
			return
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

		tmpl, err := getTemplate("dashboard",
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
			VpaData      summary.Summary
			InsightsHost string
		}{
			VpaData:      vpaData,
			InsightsHost: opts.insightsHost,
		}

		writeTemplate(tmpl, opts, &data, w)
	})
}

func calculateContainerCost(costPerCPUFloat float64, costPerGBFloat float64, c summary.ContainerSummary) float64 {
	var memCost, cpuCost float64
	if c.Limits != nil {
		cpuCost = costPerCPUFloat * (float64(c.Requests.Cpu().Value() + c.Limits.Cpu().Value())) / 2
		memCost = costPerGBFloat * (ConvertToGB(c.Requests.Memory().Value()) + ConvertToGB(c.Limits.Memory().Value())) / 2
	} else {
		cpuCost = costPerCPUFloat * float64(c.Requests.Cpu().Value())
		memCost = costPerGBFloat * ConvertToGB(c.Requests.Memory().Value())
	}
	return toFixed(cpuCost+memCost, 4)
}

func calculateRecommendedCosts(costPerCPUFloat float64, costPerGBFloat float64, containerCost float64, c summary.ContainerSummary) (float64, float64) {
	guaranteedCpuCostRecommended := costPerCPUFloat * float64(c.Target.Cpu().Value())
	guaranteedMemCosttRecommended := costPerGBFloat * ConvertToGB(c.Target.Memory().Value())

	burstableCpuCostRecommended := costPerCPUFloat * (float64(c.LowerBound.Cpu().Value() + c.UpperBound.Cpu().Value())) / 2
	burstableMemCosttRecommended := costPerGBFloat * (ConvertToGB(c.LowerBound.Memory().Value()) + ConvertToGB(c.UpperBound.Memory().Value())) / 2

	return toFixed(containerCost-(guaranteedCpuCostRecommended+guaranteedMemCosttRecommended), 4), toFixed(containerCost-(burstableCpuCostRecommended+burstableMemCosttRecommended), 4)
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
