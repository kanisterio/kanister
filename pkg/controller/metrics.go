package controller

import (
	kanistermetrics "github.com/kanisterio/kanister/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	actionSetResolutionCounterVec prometheus.CounterVec
}

const (
	ACTION_SET_COUNTER_VEC_LABEL_RES         = "resolution"
	ACTION_SET_COUNTER_VEC_LABEL_RES_SUCCESS = "success"
	ACTION_SET_COUNTER_VEC_LABEL_RES_FAILURE = "failure"
)

func getActionSetCounterVecLabels() []kanistermetrics.BoundedLabel {
	bl := make([]kanistermetrics.BoundedLabel, 1)
	bl[0] = kanistermetrics.BoundedLabel{LabelName: ACTION_SET_COUNTER_VEC_LABEL_RES,
		LabelValues: []string{ACTION_SET_COUNTER_VEC_LABEL_RES_SUCCESS, ACTION_SET_COUNTER_VEC_LABEL_RES_FAILURE}}
	return bl
}

func newMetrics(reg prometheus.Registerer) *metrics {
	actionSetCounterOpts := prometheus.CounterOpts{
		Name: "kanister_action_set_resolutions_total",
		Help: "Total number of action set resolutions",
	}
	actionSetResolutionCounterVec := kanistermetrics.InitCounterVec(reg, actionSetCounterOpts, getActionSetCounterVecLabels())
	return &metrics{actionSetResolutionCounterVec: *actionSetResolutionCounterVec}
}
