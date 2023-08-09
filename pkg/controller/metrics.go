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

package controller

import (
	"github.com/prometheus/client_golang/prometheus"

	kanistermetrics "github.com/kanisterio/kanister/pkg/metrics"
)

// metrics encapsulates all the prometheus metrics that controller
// needs to own.
type metrics struct {
	actionSetResolutionCounterVec prometheus.CounterVec
}

const (
	ACTION_SET_COUNTER_VEC_LABEL_RES         = "resolution"
	ACTION_SET_COUNTER_VEC_LABEL_RES_SUCCESS = "success"
	ACTION_SET_COUNTER_VEC_LABEL_RES_FAILURE = "failure"
)

// getActionSetCounterVecLabels builds a new BoundedLabel list to construct
// the labels permutations for the prometheus metric.
func getActionSetCounterVecLabels() []kanistermetrics.BoundedLabel {
	bl := make([]kanistermetrics.BoundedLabel, 1)
	bl[0] = kanistermetrics.BoundedLabel{LabelName: ACTION_SET_COUNTER_VEC_LABEL_RES,
		LabelValues: []string{ACTION_SET_COUNTER_VEC_LABEL_RES_SUCCESS, ACTION_SET_COUNTER_VEC_LABEL_RES_FAILURE}}
	return bl
}

// newMetrics constructs a new metrics object that encapsulates all the
// prometheus metric objects that the controller package needs to own.
func newMetrics(reg prometheus.Registerer) *metrics {
	actionSetCounterOpts := prometheus.CounterOpts{
		Name: "kanister_action_set_resolutions_total",
		Help: "Total number of action set resolutions",
	}
	actionSetResolutionCounterVec := kanistermetrics.InitCounterVec(reg, actionSetCounterOpts, getActionSetCounterVecLabels())
	return &metrics{actionSetResolutionCounterVec: *actionSetResolutionCounterVec}
}
