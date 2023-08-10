// Copyright 2023 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package metrics

import (
	"reflect"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	. "gopkg.in/check.v1"
)

type MetricsSuite struct{}

func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&MetricsSuite{})

func (m *MetricsSuite) TestGetLabelCombinations(c *C) {
	boundedLabels := make([]BoundedLabel, 3)
	boundedLabels[0] = BoundedLabel{LabelName: "operation_type", LabelValues: []string{"backup", "restore"}}
	boundedLabels[1] = BoundedLabel{LabelName: "resolution", LabelValues: []string{"success", "failure"}}
	boundedLabels[2] = BoundedLabel{LabelName: "namespace", LabelValues: []string{"kanister", "N/A"}}
	expectedPrometheusLabels := []prometheus.Labels{
		{
			"operation_type": "backup",
			"resolution":     "success",
			"namespace":      "kanister",
		},
		{
			"operation_type": "backup",
			"resolution":     "success",
			"namespace":      "N/A",
		},
		{
			"operation_type": "backup",
			"resolution":     "failure",
			"namespace":      "kanister",
		},
		{
			"operation_type": "backup",
			"resolution":     "failure",
			"namespace":      "N/A",
		},
		{
			"operation_type": "restore",
			"resolution":     "success",
			"namespace":      "kanister",
		},
		{
			"operation_type": "restore",
			"resolution":     "success",
			"namespace":      "N/A",
		},
		{
			"operation_type": "restore",
			"resolution":     "failure",
			"namespace":      "kanister",
		},
		{
			"operation_type": "restore",
			"resolution":     "failure",
			"namespace":      "N/A",
		},
	}
	receivedCombinations, err := getLabelCombinations(boundedLabels)
	c.Assert(err, IsNil)
	isEqual := reflect.DeepEqual(receivedCombinations, expectedPrometheusLabels)
	c.Assert(isEqual, Equals, true)

	boundedLabels = make([]BoundedLabel, 0)
	receivedCombinations, err = getLabelCombinations(boundedLabels)
	c.Assert(receivedCombinations, IsNil)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "invalid BoundedLabel list")

	boundedLabels = make([]BoundedLabel, 1)
	boundedLabels[0] = BoundedLabel{LabelName: "resolution", LabelValues: []string{"success", "failure"}}
	expectedPrometheusLabels = []prometheus.Labels{
		{
			"resolution": "success",
		},
		{
			"resolution": "failure",
		},
	}
	receivedCombinations, err = getLabelCombinations(boundedLabels)
	isEqual = reflect.DeepEqual(receivedCombinations, expectedPrometheusLabels)
	c.Assert(err, IsNil)
	c.Assert(isEqual, Equals, true)

	boundedLabels = make([]BoundedLabel, 1)
	boundedLabels[0] = BoundedLabel{LabelName: "resolution", LabelValues: nil}

	receivedCombinations, err = getLabelCombinations(boundedLabels)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "invalid BoundedLabel list")
	c.Assert(receivedCombinations, IsNil)

	boundedLabels = make([]BoundedLabel, 1)
	boundedLabels[0] = BoundedLabel{LabelName: "resolution", LabelValues: []string{}}
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "invalid BoundedLabel list")
	c.Assert(receivedCombinations, IsNil)
}

func (m *MetricsSuite) TestInitCounterVec(c *C) {
	boundedLabels := make([]BoundedLabel, 2)
	boundedLabels[0] = BoundedLabel{LabelName: "operation_type", LabelValues: []string{"backup", "restore"}}
	boundedLabels[1] = BoundedLabel{LabelName: "resolution", LabelValues: []string{"success", "failure"}}
	actionSetCounterOpts := prometheus.CounterOpts{
		Name: "action_set_resolutions_total",
		Help: "Total number of action set resolutions",
	}
	registry := prometheus.NewRegistry()
	metrics, err := registry.Gather()
	c.Assert(metrics, IsNil)
	c.Assert(err, IsNil)

	actionSetCounterVec := InitCounterVec(registry, actionSetCounterOpts, boundedLabels)

	metrics, err = registry.Gather()
	c.Assert(metrics, NotNil)
	c.Assert(err, IsNil)
	c.Assert(len(metrics), Equals, 1)

	expectedOperationTypes := map[string]int{"backup": 0, "restore": 0}
	expectedResolutionTypes := map[string]int{"success": 0, "failure": 0}
	for _, metric := range metrics[0].Metric {
		for _, label := range metric.Label {
			if *label.Name == "operation_type" {
				expectedOperationTypes[*label.Value] += 1
			} else if *label.Name == "resolution" {
				expectedResolutionTypes[*label.Value] += 1
			}
		}
	}
	c.Assert(expectedOperationTypes["backup"], Equals, 2)
	c.Assert(expectedOperationTypes["restore"], Equals, 2)
	c.Assert(expectedResolutionTypes["success"], Equals, 2)
	c.Assert(expectedResolutionTypes["failure"], Equals, 2)

	c.Assert(testutil.ToFloat64(actionSetCounterVec.WithLabelValues("backup", "success")), Equals, float64(0))
	c.Assert(testutil.ToFloat64(actionSetCounterVec.WithLabelValues("backup", "failure")), Equals, float64(0))
	c.Assert(testutil.ToFloat64(actionSetCounterVec.WithLabelValues("restore", "success")), Equals, float64(0))
	c.Assert(testutil.ToFloat64(actionSetCounterVec.WithLabelValues("restore", "failure")), Equals, float64(0))
}
