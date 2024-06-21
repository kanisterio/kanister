# Enhancing observability in Kanister

<!-- toc -->
- [Motivation](#motivation)
- [Scope](#scope)
- [Detailed Design](#detailed-design)
  - [Architecture](#architecture)
    - [Text description](#text-description)
  - [Low Level APIs](#low-level-apis)
    - [Metrics Package](#metrics-package)
      - [Example Initialization Steps for a new CounterVec metric](#example-initialization-steps-for-a-new-countervec-metric)
    - [Consumer Package](#consumer-package)
- [Testing](#testing)
<!-- /toc -->

This document proposes the changes required within Kanister to integrate
[Prometheus](https://github.com/prometheus/prometheus) and enhance the
observability of action sets.

## Motivation

Kanister controller already has a registered metrics endpoint `/metrics`.
There are no metrics exported other than the default Prometheus metrics that
the default handler provides. Adding metrics to track the ActionSets and
Blueprints workflow will help improve the overall observability.

To achieve this, we need to build a framework for exporting metrics from the
Kanister controller, and to start with, export some metrics to Prometheus.

This framework simplifies the common need for Prometheus counters to publish 0
values at startup for all permutations of labels and label values. This ensures
that Kanister controller restarts are recognized by Prometheus and that the
PromQL rate() and increase() functions work properly across restarts.

Some example metrics include:
ActionSets succeeded
ActionSets failed
Action duration
Phase duration, etc.


## Scope

1. Design a framework that allows us to export new Kanister metrics to
   Prometheus easily.
2. Add a few fundamental metrics related to ActionSets and Blueprints to start
   with.


## Detailed Design

### Architecture

![Alt text](images/prometheus-metrics-design.png?raw=true "Prometheus Integration Design")

#### Text description

1. The initializer of the consumer package calls `newMetrics`, a helper method
   that talks to Kanister’s metrics package. The result is a new metrics struct
   that owns all the Prometheus metrics.

2. In order to initialize all the required Prometheus metrics, the
   `new_metrics` method calls the `InitCounterVec`, `InitGaugeVec`,
   `InitHistogramVec` in the metrics package. It passes the metric names and
   the specific label names and label values as `BoundedLabels` to the metrics
   package. Once it initializes all the Prometheus metrics successfully, it
   returns a struct that wraps all the metrics for the consumer package to use.

3. The metrics package internally initializes the Prometheus metrics and
   registers them with Prometheus. If the registration fails because the specific
   metric with label header already exists, the metric will simply be returned
   to the caller. If the registration fails due to other reasons, then the
   metrics package will cause a panic, signaling programmer error. In case of
   the `CounterVec`, the `InitCounterVec` function generates all possible
   permutations of label values and initializes each counter within the
   `CounterVec` with a value of 0.

4. Once the collector is created in the metrics package, it will be returned to
   the consumer package’s `newMetrics` helper method.

5. Once the initialization of all Prometheus metrics are complete, a new
   metrics struct will be returned to the consumer package's initializer.

6. The consumer package may find it useful to implement a helper method that
   constructs a `prometheus.Labels` mapping to access a specific counter from a
   `CounterVec` and perform an increment operation.


### APIs

#### Metrics Package

```golang
// BoundedLabel is a type that represents a label and its associated
// valid values
type BoundedLabel struct {
	LabelName   string
	LabelValues []string
}
```

An example of a `BoundedLabel` is in the scenario of ActionSet resolutions.
Suppose we want to track these resolutions across different blueprints, we
would create the bounded labels in the following way:

##### BoundedLabel example

```golang
BoundedLabel{
	LabelName: "operation_type",
	LabelValues: []string{
		"backup",
		"restore",
	},
}

BoundedLabel{
	LabelName: "action_set_resolution",
	LabelValues: []string{
		"success",
		"failure",
	},
}
```

##### Initialization methods

```golang

// InitCounterVec initializes and registers the counter metrics vector. It
// takes a list of BoundedLabel objects - if any label value or label name is
// nil, then this method will panic. Based on the combinations returned by
// generateCombinations, it will set each counter value to 0.
// If a nil counter is returned during registration, the method will panic.
func InitCounterVec(r prometheus.Registerer, opts prometheus.CounterOpts, boundedLabels []BoundedLabel) *prometheus.CounterVec

// InitGaugeVec initializes the gauge metrics vector. It takes a list of
// BoundedLabels, but the LabelValue field of each BoundedLabel will be
// ignored. If a nil counter is returned during registration, the method will
// panic.
func InitGaugeVec(r prometheus.Registerer, opts prometheus.CounterOpts, boundedLabels []BoundedLabel) *prometheus.GaugeVec

// InitHistogramVec initializes the histogram metrics vector. It takes a list
// of BoundedLabels, but the LabelValue field of each BoundedLabel will be
// ignored. If a nil counter is returned during registration, the method will
// panic.
func InitHistogramVec(r prometheus.Registerer, opts prometheus.CounterOpts, boundedLabels []BoundedLabel) *prometheus.HistogramVec

// InitCounter initializes a new counter.
// If a nil counter is returned during registration, the method will panic.
func InitCounter(r prometheus.Registerer, opts prometheus.CounterOpts) prometheus.Counter

// InitGauge initializes a new gauge.
// If a nil counter is returned during registration, the method will panic.
func InitGauge(r prometheus.Registerer, opts prometheus.GaugeOpts) prometheus.Gauge

// InitHistogram initializes a new histogram.
// If a nil counter is returned during registration, the method will panic.
func InitHistogram(r prometheus.Registerer, opts prometheus.HistogramOpts) prometheus.Histogram
```

##### Example Initialization Steps for a new CounterVec metric

1. Initialize a new `CounterVec` with relevant options and label names.

2. Attempt to register the new `CounterVec`.
   * If successful,
      * Generate combinations of label names.
      * Create counters for each combination and set the counter to 0.

   * If not successful, check if the error is an `AlreadyRegisteredError`.
      * If yes, return the `CounterVec` and ignore the error.
      * If no, then panic, signalling programmer error.

3. If received a `CounterVec` from registration, it is guaranteed that the
   registration is successful.

#### Consumer Package

The below example will walk through how a consumer package will be integrated
with the metrics package:

Each consumer package in Kanister will have a main struct and a `metrics.go`
file.

An example of this would be the controller package:

controller/controller.go

```golang
type Controller struct {
	config           *rest.Config
	crClient         versioned.Interface
	clientset        kubernetes.Interface
	dynClient        dynamic.Interface
	osClient         osversioned.Interface
	recorder         record.EventRecorder
	actionSetTombMap sync.Map
	metrics          *metrics // add a new member to the existing struct
}
```

```golang
// New creates a controller for watching Kanister custom resources created.
func New(c *rest.Config) *Controller {
	return &Controller{
		config:  c,
		metrics: newMetrics(prometheus.DefaultRegistry), // this helper method call will be made during init
	}
}
```


controller/metrics.go

```golang
const (
	ActionSetCounterVecLabelRes     = "resolution"
	ACTION_SET_COUNTER_VEC_LABEL_OP_TYPE = "operation_type"
)

type metrics struct {
	ActionSetCounterVec *prometheus.CounterVec
}

// getActionSetCounterVecLabels is a helper method to construct the correct
// "LabelHeaders":"LabelValues" mapping to ensure type safety.
func getActionSetCounterVecLabels() []kanistermetrics.BoundedLabel {
	bl := make([]kanistermetrics.BoundedLabel, 2)
	bl[0] = kanistermetrics.BoundedLabel{
		LabelName:   ActionSetCounterVecLabelRes,
		LabelValues: []string{"success", "failure"},
	}
	bl[1] = kanistermetrics.BoundedLabel{
		LabelName:   ACTION_SET_COUNTER_VEC_LABEL_BLUEPRINT,
		LabelValues: []string{"backup", "restore"},
	}
	return bl
}


// constructActionSetCounterVecLabels is a helper method to construct the
// labels correctly.
func constructActionSetCounterVecLabels(operation_type string, resolution string) prometheus.Labels {
	return prometheus.Labels{
		ACTION_SET_COUNTER_VEC_LABEL_OP_TYPE: operation_type,
		ActionSetCounterVecLabelRes:     resolution,
	}
}

// newMetrics is a helper method to create a Metrics interface.
func newMetrics(gatherer prometheus.Gatherer) *metrics {
	actionSetCounterOpts := prometheus.CounterOpts{
		Name: "action_set_resolutions_total",
		Help: "Total number of action set resolutions",
	}
	actionSetCounterVec := kanistermetrics.InitCounterVec(
		gatherer,
		actionSetCounterOpts,
		getActionSetCounterVecLabels(),
	)
	return &metrics{ActionSetCounterVec: actionSetCounterVec}
}
```

The below example will show how the above created `ActionSetCounterVec` will
be incremented in a method:

```golang
func (c *Controller) handleActionSet(ctx context.Context) {
	c.metrics.ActionSetCounterVec.With(constructActionSetCounterVecLabels("backup", "success")).Inc()
}
```

Alternatively, one can also directly call the Prometheus API with positional
arguments:

```golang
func (c *Controller) handleActionSet(ctx context.Context) {
	c.metrics.ActionSetCounterVec.WithLabelValues("backup", "success").Inc()
}
```


## Testing

1. The testing will include manual testing of whether the metrics added are
   successfully getting exported to Kanister.

2. The interfaces listed above in the metrics package, apart from
   `InitCounterVec` and `generateLabelCombinations`, will not be unit-tested,
   since they would be testing the behavior of the Prometheus API itself, which
   breaks the chain of trust principle with dependencies in unit testing.
   `InitCounterVec` will be unit tested using the
   [test_util](https://pkg.go.dev/github.com/prometheus/client_golang/prometheus/testutil)
   package in Prometheus.

3. Integration tests will be added for code that exports new metrics, to ensure
   that the behavior of exporting metrics is correct.