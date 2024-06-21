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
	"errors"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"gonum.org/v1/gonum/stat/combin"

	"github.com/kanisterio/kanister/pkg/log"
)

// BoundedLabel is a type that represents a label and its associated valid values.
type BoundedLabel struct {
	LabelName   string
	LabelValues []string
}

// getLabelNames gets the all the `LabelName` fields from each BoundedLabel struct.
func getLabelNames(bl []BoundedLabel) []string {
	ln := make([]string, len(bl))
	for idx, l := range bl {
		ln[idx] = l.LabelName
	}
	return ln
}

// verifyBoundedLabels checks if the BoundedLabel list is valid
// returns true if valid, and false if invalid.
func verifyBoundedLabels(bl []BoundedLabel) bool {
	if len(bl) == 0 {
		return false
	}
	for _, l := range bl {
		if len(l.LabelValues) == 0 {
			return false
		}
	}
	return true
}

// getLabelCombinations takes a slice of BoundedLabel elements and
// returns a list of permutations of possible label permutations.
func getLabelCombinations(bl []BoundedLabel) ([]prometheus.Labels, error) {
	/*
		Considering the following example - If we have two BoundedLabel elements:
		BoundedLabel{
		  LabelName: "operation_type"
		  LabelValues: ["backup", "restore"]
		}
		BoundedLabel{
		  LabelName: "action_set_resolution"
		  LabelValues: ["success", "failure"]
		}
		The following code generates the permutation list:
		[ {"operation_type": "backup", "action_set_resolution": "success"},
		{"operation_type": "backup", "action_set_resolution": "failure"},
		{"operation_type": "restore", "action_set_resolution": "success"},
		{"operation_type": "restore", "action_set_resolution": "failure"}]
	*/
	if !verifyBoundedLabels(bl) {
		return nil, errors.New("invalid BoundedLabel list")
	}
	labelLens := make([]int, len(bl))
	for idx, l := range bl {
		labelLens[idx] = len(l.LabelValues)
	}
	idxPermutations := combin.Cartesian(labelLens)

	resultPrometheusLabels := make([]prometheus.Labels, len(idxPermutations))

	// generate the actual label permutations from the index permutations
	// obtained
	for idx, perm := range idxPermutations {
		labelSet := make(prometheus.Labels)
		for idx, p := range perm {
			labelSet[bl[idx].LabelName] = bl[idx].LabelValues[p]
		}
		resultPrometheusLabels[idx] = labelSet
	}
	return resultPrometheusLabels, nil
}

// setDefaultCounterWithLabels initializes all the counters within a CounterVec
// and sets them to 0.
func setDefaultCounterWithLabels(cv *prometheus.CounterVec, l []prometheus.Labels) {
	for _, c := range l {
		cv.With(c)
	}
}

// InitCounterVec initializes and registers the counter metrics vector. It
// takes a list of BoundedLabel objects - if any label value or label name is
// nil, then this method will panic. Based on the combinations returned by
// generateCombinations, it will set each counter value to 0.
// If a nil counter is returned during the registration, the method will panic.
func InitCounterVec(r prometheus.Registerer, opts prometheus.CounterOpts, bl []BoundedLabel) *prometheus.CounterVec {
	labels := getLabelNames(bl)
	v := prometheus.NewCounterVec(opts, labels)
	gv := registerCounterVec(r, v)
	combinations, err := getLabelCombinations(bl)
	if err != nil {
		panic(fmt.Sprintf("failed to register CounterVec. error: %v", err))
	}
	setDefaultCounterWithLabels(gv, combinations)
	return gv
}

// InitGaugeVec initializes the gauge metrics vector. It takes a list of
// BoundedLabels, but the LabelValue field of each BoundedLabel will be
// ignored. The method panics if there are any errors (except for
// AlreadyRegisteredError) during the registration of the metric.
func InitGaugeVec(r prometheus.Registerer, opts prometheus.GaugeOpts, bl []BoundedLabel) *prometheus.GaugeVec {
	labels := getLabelNames(bl)
	v := prometheus.NewGaugeVec(opts, labels)
	gv := registerGaugeVec(r, v)
	return gv
}

// InitHistogramVec initializes the histogram metrics vector. It takes a list
// of BoundedLabels, but the LabelValue field of each BoundedLabel will be
// ignored. The method panics if there are any errors (except for
// AlreadyRegisteredError) during the registration of the metric.
func InitHistogramVec(r prometheus.Registerer, opts prometheus.HistogramOpts, bl []BoundedLabel) *prometheus.HistogramVec {
	labels := getLabelNames(bl)
	v := prometheus.NewHistogramVec(opts, labels)
	h := registerHistogramVec(r, v)
	return h
}

// InitCounter initializes a new counter. The method panics if there are any
// errors (except for AlreadyRegisteredError) during the registration of the
// metric.
func InitCounter(r prometheus.Registerer, opts prometheus.CounterOpts) prometheus.Counter {
	c := prometheus.NewCounter(opts)
	rc := registerCounter(r, c)
	return rc
}

// InitGauge initializes a new gauge metric. The method panics if there are any
// errors (except for AlreadyRegisteredError) during the registration of the
// metric.
func InitGauge(r prometheus.Registerer, opts prometheus.GaugeOpts) prometheus.Gauge {
	g := prometheus.NewGauge(opts)
	rg := registerGauge(r, g)
	return rg
}

// InitHistogram initializes a new histogram metric. The method panics if there
// are any errors (except for AlreadyRegisteredError) during the registration
// of the metric.
func InitHistogram(r prometheus.Registerer, opts prometheus.HistogramOpts) prometheus.Histogram {
	h := prometheus.NewHistogram(opts)
	rh := registerHistogram(r, h)
	return rh
}

// registerCounterVec registers the CounterVec with the provided Registerer. It
// panics if the type check fails.
func registerCounterVec(r prometheus.Registerer, g *prometheus.CounterVec) *prometheus.CounterVec {
	c := registerMetricOrDie(r, g)
	gv, ok := c.(*prometheus.CounterVec)
	if !ok {
		panic("failed type check for CounterVec")
	}
	return gv
}

// registerHistogramVec registers the HistogramVec with the provided
// Registerer. It panics if the type check fails.
func registerHistogramVec(r prometheus.Registerer, h *prometheus.HistogramVec) *prometheus.HistogramVec {
	c := registerMetricOrDie(r, h)
	v, ok := c.(*prometheus.HistogramVec)
	if !ok {
		panic("failed type check for HistogramVec")
	}
	return v
}

// registerGaugeVec registers the GaugeVec with the provided Registerer. It
// panics if the type check fails.
func registerGaugeVec(r prometheus.Registerer, g *prometheus.GaugeVec) *prometheus.GaugeVec {
	c := registerMetricOrDie(r, g)
	gv, ok := c.(*prometheus.GaugeVec)
	if !ok {
		panic("failed type check for GaugeVec")
	}
	return gv
}

// registerCounter registers the Counter with the provided Registerer. It
// panics if the type check fails.
func registerCounter(r prometheus.Registerer, ctr prometheus.Counter) prometheus.Counter {
	c := registerMetricOrDie(r, ctr)
	rg, ok := c.(prometheus.Counter)
	if !ok {
		panic("failed type check for Counter")
	}
	return rg
}

// registerHistogram registers the Histogram with the provided Registerer. It
// panics if the type check fails.
func registerHistogram(r prometheus.Registerer, g prometheus.Histogram) prometheus.Histogram {
	c := registerMetricOrDie(r, g)
	rg, ok := c.(prometheus.Histogram)
	if !ok {
		panic("failed type check for Histogram")
	}
	return rg
}

// registerGauge registers the Gauge with the provided Registerer. It panics if
// the type check fails.
func registerGauge(r prometheus.Registerer, g prometheus.Gauge) prometheus.Gauge {
	c := registerMetricOrDie(r, g)
	rg, ok := c.(prometheus.Gauge)
	if !ok {
		panic("failed type check for Gauge")
	}
	return rg
}

// registerMetricOrDie is a helper to register a metric and log registration
// errors. If the metric already exists, then it will be logged, and the metric
// is returned. For other errors, the method panics.
func registerMetricOrDie(r prometheus.Registerer, c prometheus.Collector) prometheus.Collector {
	if err := r.Register(c); err != nil {
		are := prometheus.AlreadyRegisteredError{}
		if !errors.As(err, &are) {
			panic(fmt.Sprintf("failed to register metric. error: %v", err))
		}
		// Use already registered metric
		log.Debug().Print("Metric already registered")
		return are.ExistingCollector
	}
	return c
}
