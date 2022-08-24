package metrics

import (
	"errors"
	"fmt"

	"github.com/kanisterio/kanister/pkg/log"
	"github.com/prometheus/client_golang/prometheus"
)

var counterVecs = make(map[MetricType]*prometheus.CounterVec)

// Initialize a Prometheus CounterVec for one metric and register it
func initCounterVec(r prometheus.Registerer, t MetricType) (*prometheus.CounterVec, error) {
	metricTypeOpts, ok := MetricCounterOpts[t]

	if !ok {
		return nil, fmt.Errorf("Event type %s is not defined", t)
	}

	opts := prometheus.CounterOpts{
		Name: string(t),
		Help: metricTypeOpts.Help,
	}
	counterVec := prometheus.NewCounterVec(opts, metricTypeOpts.LabelNames)

	err := r.Register(counterVec)
	if err != nil {
		return nil, fmt.Errorf("%s not registered: %s ", t, err)
	}
	var alreadyRegisteredErr prometheus.AlreadyRegisteredError
	if errors.As(err, &alreadyRegisteredErr) {
		counterVec = alreadyRegisteredErr.ExistingCollector.(*prometheus.CounterVec)
	} else if err != nil {
		return nil, fmt.Errorf("Error registering Counter Vecs : %s ", err)
	}

	return counterVec, nil
}

// Initialize all the Counter Vecs and save it in a map
func InitAllCounterVecs(r prometheus.Registerer) map[MetricType]*prometheus.CounterVec {
	for metricType := range MetricCounterOpts {
		cv, err := initCounterVec(r, metricType)
		if err != nil {
			log.WithError(err).Print("Failed to register metric %s")
			return nil
		}
		counterVecs[metricType] = cv
	}
	return counterVecs
}

// Increment a Counter Vec metric
func IncrementCounterVec(e Event) error {
	if counterVecs[e.eventType] == nil {
		return fmt.Errorf("%s Event Type not found", e.eventType)
	}
	cv := counterVecs[e.eventType].With(e.labels)
	if cv == nil {
		return fmt.Errorf("%s Labels for %s Event Type not found", e.labels, e.eventType)
	}
	cv.Inc()
	return nil
}
