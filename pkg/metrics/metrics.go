package metrics

import (
	"fmt"

	"github.com/kanisterio/kanister/pkg/log"
	"github.com/prometheus/client_golang/prometheus"
)

var counterVecs map[MetricType]*prometheus.CounterVec

// Initialize a Prometheus CounterVec for one metric and register it
func initCounterVec(r prometheus.Registerer, t MetricType) (*prometheus.CounterVec, error) {
	metricTypeOpts, ok := MetricTypeOpts[t]

	if !ok {
		panic(fmt.Sprintf("Event type %s is not defined", t))
	}

	opts := prometheus.CounterOpts{
		Name: string(t),
		Help: metricTypeOpts.Help,
	}
	counterVec := prometheus.NewCounterVec(opts, metricTypeOpts.LabelNames)

	if err := r.Register(counterVec); err != nil {
		return nil, fmt.Errorf("%s not registered: %s ", t, err)
	}

	return counterVec, nil
}

// Initialize a Prometheus GaugeVec for one metric and register it
// func initGaugeVec(r prometheus.Registerer, t MetricType) (*prometheus.GaugeVec, error) {
// 	metricTypeOpts, ok := MetricTypeOpts[t]

// 	if !ok {
// 		panic(fmt.Sprintf("Event type %s is not defined", t))
// 	}

// 	opts := prometheus.GaugeOpts{
// 		Name: string(t),
// 		Help: metricTypeOpts.Help,
// 	}
// 	gaugeVec := prometheus.NewGaugeVec(opts, metricTypeOpts.LabelNames)

// 	if err := r.Register(gaugeVec); err != nil {
// 		return nil, fmt.Errorf("%s not registered: %s ", t, err)
// 	}

// 	return gaugeVec, nil
// }

func InitAllCounterVecs(r prometheus.Registerer) {
	for metricType := range MetricTypeOpts {
		cv, err := initCounterVec(r, metricType)
		if err != nil {
			log.WithError(err).Print("Failed to register metric %s")
			return
		}
		counterVecs[metricType] = cv
	}
}

// Increment a Counter Vec metric
func IncrementCounterVec(e Event) {
	counterVecs[e.eventType].With(e.labels).Inc()
}

// on status change, create list of events to increment metrics for
