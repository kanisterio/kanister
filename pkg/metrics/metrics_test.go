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
	expectedPrometheusLabels := []prometheus.Labels{{
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
	receivedCombinations := getLabelCombinations(boundedLabels)
	isEqual := reflect.DeepEqual(receivedCombinations, expectedPrometheusLabels)
	c.Assert(isEqual, Equals, true)

	boundedLabels = make([]BoundedLabel, 0)
	expectedPrometheusLabels = []prometheus.Labels{}
	receivedCombinations = getLabelCombinations(boundedLabels)
	isEqual = reflect.DeepEqual(receivedCombinations, expectedPrometheusLabels)
	c.Assert(isEqual, Equals, true)

	boundedLabels = make([]BoundedLabel, 1)
	boundedLabels[0] = BoundedLabel{LabelName: "resolution", LabelValues: []string{"success", "failure"}}
	expectedPrometheusLabels = []prometheus.Labels{{
		"resolution": "success",
	}, {
		"resolution": "failure",
	}}
	receivedCombinations = getLabelCombinations(boundedLabels)
	isEqual = reflect.DeepEqual(receivedCombinations, expectedPrometheusLabels)
	c.Assert(isEqual, Equals, true)
}

func (m *MetricsSuite) TestInitCounterVec(c *C) {
	boundedLabels := make([]BoundedLabel, 2)
	boundedLabels[0] = BoundedLabel{LabelName: "operation_type", LabelValues: []string{"backup", "restore"}}
	boundedLabels[1] = BoundedLabel{LabelName: "resolution", LabelValues: []string{"success", "failure"}}
	actionSetCounterOpts := prometheus.CounterOpts{
		Name: "action_set_resolutions_total",
		Help: "Total number of action set resolutions",
	}
	actionSetCounterVec := InitCounterVec(prometheus.DefaultRegisterer, actionSetCounterOpts, boundedLabels)
	c.Assert(float64(0), Equals, testutil.ToFloat64(actionSetCounterVec.WithLabelValues("backup", "success")))
	c.Assert(float64(0), Equals, testutil.ToFloat64(actionSetCounterVec.WithLabelValues("backup", "failure")))
	c.Assert(float64(0), Equals, testutil.ToFloat64(actionSetCounterVec.WithLabelValues("restore", "success")))
	c.Assert(float64(0), Equals, testutil.ToFloat64(actionSetCounterVec.WithLabelValues("restore", "failure")))

	actionSetCounterVec.WithLabelValues("backup", "success").Inc()

	actionSetCounterVec = InitCounterVec(prometheus.DefaultRegisterer, actionSetCounterOpts, boundedLabels)
	c.Assert(float64(1), Equals, testutil.ToFloat64(actionSetCounterVec.WithLabelValues("backup", "success")))
	c.Assert(float64(0), Equals, testutil.ToFloat64(actionSetCounterVec.WithLabelValues("backup", "failure")))
	c.Assert(float64(0), Equals, testutil.ToFloat64(actionSetCounterVec.WithLabelValues("restore", "success")))
	c.Assert(float64(0), Equals, testutil.ToFloat64(actionSetCounterVec.WithLabelValues("restore", "failure")))
}
