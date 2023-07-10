package metrics

import (
	"errors"
	"fmt"

	"github.com/kanisterio/kanister/pkg/log"
	"github.com/prometheus/client_golang/prometheus"
)

// BoundedLabel is a type that represents a label and its associated
// valid values
type BoundedLabel struct {
	LabelName   string
	LabelValues []string
}

// getLabelNames extracts the all the `LabelName` fields from each BoundedLabel struct
func getLabelNames(bl []BoundedLabel) []string {
	ln := make([]string, 0)
	for _, l := range bl {
		ln = append(ln, l.LabelName)
	}
	return ln
}

// generateLabelCombinations uses a backtracking approach to generate
// a list of label permutations
func generateLabelCombinations(bl []BoundedLabel, labelIndex int,
	workingSlice [][]string, resultCombinations *[][][]string) {
	if labelIndex >= len(bl) {
		if len(workingSlice) == 0 {
			return
		}
		newCombination := append([][]string(nil), workingSlice...)
		*resultCombinations = append(*resultCombinations, newCombination)
		return
	}
	for i := 0; i < len(bl[labelIndex].LabelValues); i++ {
		workingSlice = append(workingSlice, []string{bl[labelIndex].LabelName, bl[labelIndex].LabelValues[i]})
		generateLabelCombinations(bl, labelIndex+1, workingSlice, resultCombinations)
		workingSlice = workingSlice[:len(workingSlice)-1]
	}
}

// getLabelCombinations takes a slice of BoundedLabel elements and
// returns a list of permutations of possible label permutations.
func getLabelCombinations(bl []BoundedLabel) []prometheus.Labels {
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
	resultCombinations := make([][][]string, 0)
	labelSlice := make([][]string, 0)
	generateLabelCombinations(bl, 0, labelSlice, &resultCombinations)
	resultPrometheusLabels := make([]prometheus.Labels, 0)
	for _, c := range resultCombinations {
		labelSet := make(prometheus.Labels)
		for _, l := range c {
			labelSet[l[0]] = l[1]
		}
		resultPrometheusLabels = append(resultPrometheusLabels, labelSet)
	}
	return resultPrometheusLabels
}

func setDefaultCounterWithLabels(cv *prometheus.CounterVec, l []prometheus.Labels) {
	for _, c := range l {
		cv.With(c).Add(0)
	}
}

// InitCounterVec initializes and registers the counter metrics vector.
// Based on the combinations returned by generateCombinations, it will set each counter value to 0.
// If a nil counter is returned during registeration, the method will
// panic
func InitCounterVec(r prometheus.Registerer, opts prometheus.CounterOpts, bl []BoundedLabel) *prometheus.CounterVec {
	labels := getLabelNames(bl)
	v := prometheus.NewCounterVec(opts, labels)
	gv, err := registerCounterVec(r, v)
	if err != nil {
		panic(fmt.Sprintf("failed to register CounterVec. error: %v", err))
	}
	combinations := getLabelCombinations(bl)
	setDefaultCounterWithLabels(gv, combinations)
	return gv
}

// InitGaugeVec initializes the gauge metrics vector.
// If a nil counter is returned during registeration, the method will
// panic
func InitGaugeVec(r prometheus.Registerer, opts prometheus.GaugeOpts, labels []string) *prometheus.GaugeVec {
	v := prometheus.NewGaugeVec(opts, labels)
	gv, err := registerGaugeVec(r, v)
	if err != nil {
		panic(fmt.Sprintf("failed to register GaugeVec. error: %v", err))
	}
	return gv
}

// InitHistogramVec initializes the histogram metrics vector
// If a nil counter is returned during registeration, the method will
// panic
func InitHistogramVec(r prometheus.Registerer, opts prometheus.HistogramOpts, labels []string) *prometheus.HistogramVec {
	v := prometheus.NewHistogramVec(opts, labels)
	h, err := registerHistogramVec(r, v)
	if err != nil {
		panic(fmt.Sprintf("failed to register HistogramVec. error: %v", err))
	}
	return h
}

// InitCounter initializes a new counter.
// If a nil counter is returned during registeration, the method will
// panic
func InitCounter(r prometheus.Registerer, opts prometheus.CounterOpts) prometheus.Counter {
	c := prometheus.NewCounter(opts)
	rc, err := registerCounter(r, c)
	if err != nil {
		panic(fmt.Sprintf("failed to register counter. error: %v", err))
	}
	return rc
}

// InitGauge initializes the histogram metrics vector
// If a nil counter is returned during registeration, the method will
// panic
func InitGauge(r prometheus.Registerer, opts prometheus.GaugeOpts) prometheus.Gauge {
	g := prometheus.NewGauge(opts)
	rg, err := registerGauge(r, g)
	if err != nil {
		panic(fmt.Sprintf("failed to register gauge. error: %v", err))
	}
	return rg
}

// InitHistogram initializes a new histogram.
// If a nil counter is returned during registeration, the method will
// panic
func InitHistogram(r prometheus.Registerer, opts prometheus.HistogramOpts) prometheus.Histogram {
	h := prometheus.NewHistogram(opts)

	rh, err := registerHistogram(r, h)
	if err != nil {
		panic(fmt.Sprintf("failed to register histogram. error: %v", err))
	}

	return rh
}

// registerCounterVec registers the CounterVec with the provided Registerer. If the
// CounterVec has already been registered, the existing metric will be returned.
func registerCounterVec(r prometheus.Registerer, g *prometheus.CounterVec) (*prometheus.CounterVec, error) {
	c, err := registerCollector(r, g)
	if err != nil {
		return nil, err
	}
	if gv, ok := c.(*prometheus.CounterVec); ok {
		return gv, nil
	}
	return nil, errors.New("failed to register counter vec")
}

// registerHistogramVec registers the Histogram with the provided Registerer. If the
// HistogramVec has already been registered, the existing metric will be returned.
func registerHistogramVec(r prometheus.Registerer, h *prometheus.HistogramVec) (*prometheus.HistogramVec, error) {
	c, err := registerCollector(r, h)
	if err != nil {
		return nil, err
	}
	if v, ok := c.(*prometheus.HistogramVec); ok {
		return v, nil
	}
	return nil, errors.New("failed to register historgram vec")
}

// registerGaugeVec registers the GaugeVec with the provided Registerer. If the
// GaugeVec has already been registered, the existing GaugeVec will be returned.
func registerGaugeVec(r prometheus.Registerer, g *prometheus.GaugeVec) (*prometheus.GaugeVec, error) {
	c, err := registerCollector(r, g)
	if err != nil {
		return nil, err
	}
	if gv, ok := c.(*prometheus.GaugeVec); ok {
		return gv, nil
	}
	return nil, errors.New("failed to register gauge vec")
}

// registerCounter registers the Counter with the provided Registerer. If the
// counter has already been registered, the existing metric will be returned.
func registerCounter(r prometheus.Registerer, cntr prometheus.Counter) (prometheus.Counter, error) {
	c, err := registerCollector(r, cntr)
	if err != nil {
		return nil, err
	}
	if rc, ok := c.(prometheus.Counter); ok {
		return rc, nil
	}
	return nil, errors.New("failed to register counter")
}

// registerHistogram registers the Histogram with the provided Registerer. If the
// histogram has already been registered, the existing metric will be returned.
func registerHistogram(r prometheus.Registerer, h prometheus.Histogram) (prometheus.Histogram, error) {
	c, err := registerCollector(r, h)
	if err != nil {
		return nil, err
	}

	if rh, ok := c.(prometheus.Histogram); ok {
		return rh, nil
	}

	return nil, errors.New("failed to register histogram")
}

// registerGauge registers the Gauge with the provided Registerer. If the
// gauge has already been registered, the existing metric will be returned.
func registerGauge(r prometheus.Registerer, g prometheus.Gauge) (prometheus.Gauge, error) {
	c, err := registerCollector(r, g)
	if err != nil {
		return nil, err
	}
	if rg, ok := c.(prometheus.Gauge); ok {
		return rg, nil
	}
	return nil, errors.New("failed to register gauge")
}

// registerCollector is an helper to register a metric and log registration errors
func registerCollector(r prometheus.Registerer, c prometheus.Collector) (prometheus.Collector, error) {
	if err := r.Register(c); err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			// Use already registered metric
			log.Debug().Print("Metric already registered")
			return are.ExistingCollector, nil
		}
		return nil, err
	}
	return c, nil
}
