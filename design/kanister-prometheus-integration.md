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
[Prometheus](https://github.com/prometheus/prometheus) and enhance the observability 
of action sets. 

## Motivation
Kanister controller already has a registered metrics endpoint `/metrics`. There are no metrics exported other than the default Prometheus metrics that the default handler provides. Adding metrics to track the 
ActionSets and Blueprints workflow will help improve the overall observability. 

To achieve this, we need to build a framework for exporting metrics from the Kanister controller, and to 
start with, export some metrics to Prometheus. 

Some example metrics include:
ActionSets succeeded
ActionSets failed
Action duration
Phase duration, etc.


## Scope
1. Design a framework that allows us to export new
Kanister metrics to Prometheus easily. 
2. Add a few fundamental metrics related to ActionSets and Blueprints to start with.


## Detailed Design

### Architecture

![Alt text](Metrics_design.png?raw=true "Prometheus Integration Design")

#### Text description
1. The initializer of the consumer package calls new_metrics, a helper method that talks to Kanister’s metrics package. The result is a new metrics struct that owns all the prometheus collectors.

2. In order to initialize all the required prometheus collectors, the new_metrics method calls the InitCounterVec, InitGaugeVec, InitHistogramVec in the metrics package. It passes the metric names and the specific label names and label values, aka, BoundedLabels, to the metrics package. Once it initializes all the prometheus collectors successfully, it returns an struct that wraps all the collectors, which the consumer package can then use. 

3. The metrics package internally attempts to initialize the collectors and register the specific collectors with prometheus. If the registration fails because the specific metric with label header already exists, the collector will simply be returned to the caller. If the registration fails due to other reasons, then the metrics package will cause a panic, signalling programmer error. In case of the CounterVec, the InitCounterVec attempts to generate all possible permutations of label values and sets each counter 
within the CounterVec to 0. 

4. Once the collector is created in the metrics package, it will be returned to the consumer package’s new_metrics helper method.

5. Once the initialization of all collectors are complete, a new metrics struct will be returned to the consumer’s package initializer. 

6. Suppose the consumer package wants to increment a specific counter in a counter vec, it constructs a prometheus.Labels mapping using a helper method to retrieve the specific counter from the counter vec and performs an increment operation. 



### Low Level APIs

#### Metrics Package



```golang
// BoundedLabel is a type that represents a label and its associated 
// valid values
type BoundedLabel struct {
	LabelName   string
	LabelValues []string
}
```
An example of a BoundedLabel is in the scenario of ActionSet resolutions.
Suppose we want to track these resolutions across different blueprints, 
we would create the bounded labels in the following way:
```golang
BoundedLabel{
  LabelName: "operation_type"
  LabelValues: ["backup", "restore"]
}

BoundedLabel{
  LabelName: "action_set_resolution"
  LabelValues: ["success", "failure"]
}
``` 

```golang

// generateCombinations generates a list of combinations of labels values, 
// which can be used by InitCounterVec to initialize all the counters within
// the CounterVec to 0

// For instance, in the above "ActionSet resolutions" example, we will
// generate the following combinations:
// [ {"operation_type": "backup", "action_set_resolution": "success"}, 
// {"operation_type": "backup", "action_set_resolution": "failure"},
// {"operation_type": "restore", "action_set_resolution": "success"}, 
// {"operation_type": "restore", "action_set_resolution": "failure"}]
func generateLabelCombinations([] BoundedLabel) []prometheus.Labels
 

// InitCounterVec initializes and registers the counter metrics vector.
// Based on the combinations returned by generateCombinations, it will set each counter value to 0.
// If a nil counter is returned during registeration, the method will
// panic
func InitCounterVec(r prometheus.Registerer, opts prometheus.CounterOpts, boundedLabels []BoundedLabel) *prometheus.CounterVec

// InitGaugeVec initializes the gauge metrics vector.
// If a nil counter is returned during registeration, the method will
// panic
func InitGaugeVec(r prometheus.Registerer, opts prometheus.CounterOpts, labels []string) *prometheus.GaugeVec

// InitGauge initializes the histogram metrics vector
// If a nil counter is returned during registeration, the method will
// panic
func InitHistogramVec(r prometheus.Registerer, opts prometheus.CounterOpts, labels []string) *prometheus.HistogramVec

// InitCounter initializes a new counter.
// If a nil counter is returned during registeration, the method will
// panic
func InitCounter(r prometheus.Registerer, opts prometheus.CounterOpts) prometheus.Counter

// InitGauge initializes a new gauge.
// If a nil counter is returned during registeration, the method will
// panic
func InitGauge(r prometheus.Registerer, opts prometheus.GaugeOpts) prometheus.Gauge

// InitHistogram initializes a new histogram.
// If a nil counter is returned during registeration, the method will
// panic
func InitHistogram(r prometheus.Registerer, opts prometheus.HistogramOpts) prometheus.Histogram

// registerCollector is an helper to register a metric and log registration errors
// If a nil counter is returned during registeration, the method will
// panic
func registerCounterVec(r prometheus.Registerer, g *prometheus.CounterVec) (*prometheus.CounterVec, error) 

// registerHistogramVec registers the Histogram vector metrics.
// If a nil counter is returned during registeration, the method will
// panic
func registerHistogramVec(r prometheus.Registerer, g *prometheus.HistogramVec) (*prometheus.HistogramVec, error) 

// registerGaugeVec registers the Gauge vector metrics.
// If a nil counter is returned during registeration, the method will
// panic
func registerGaugeVec(r prometheus.Registerer, g *prometheus.GaugeVec) (*prometheus.GaugeVec, error) 

// registerGauge registers the Gauge with the provided Registerer. If the
// gauge has already been registered, the existing metric will be returned.
func registerGauge(r prometheus.Registerer, g *prometheus.Gauge) (*prometheus.Gauge, error) 

// registerCounter registers the Counter with the provided Registerer. If the
// counter has already been registered, the existing metric will be returned.
func registerCounter(r prometheus.Registerer, g *prometheus.Counter) (*prometheus.Counter, error) 

// registerHistogram registers the Histogram with the provided Registerer. If the
// histogram has already been registered, the existing metric will be returned.
func registerHistogram(r prometheus.Registerer, g *prometheus.Counter) (*prometheus.Histogram, error) 

// registerCollector is an helper to register a metric and log registration errors
func registerCollector(r prometheus.Registerer, c prometheus.Collector) (prometheus.Collector, error)
```

##### Example Initialization Steps for a new CounterVec metric
1. Initialize a new CounterVec with relevant options and label names 

2. Attempt to register the new CounterVec

   a. If successful,

       i. Generate combinations of label names

       ii. Create counters for each combination and set the counter 
           to 0.

   b. If not successful, check if the error is an AlreadyRegisteredError

       i. If yes, return the CounterVec and ignore the error

       ii. If no, return a nil CounterVec and the received error. 

3. If received a nil CounterVec from regitration, interrupt with a panic, because an interrupt would suggest a failure in the created CounterVec, which should be fixed by the programmer. 
#### Consumer Package

The below example change will walk through how a consumer package will be integrated with the metrics package:

Each consumer package in Kanister will have a main struct and a "metrics.go” file . An example of this would be the controller package: 

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
// New create controller for watching kanister custom resources created
func New(c *rest.Config) *Controller {
	return &Controller{
		config:  c,
		metrics: newMetrics(), // this helper method call will be made during init
	}
}
```


controller/metrics.go

```golang
const (
    ACTION_SET_COUNTER_VEC_LABEL_RES = "resolution"
    ACTION_SET_COUNTER_VEC_LABEL_OP_TYPE = "operation_type"
)

type Metrics struct {
    ActionSetCounterVec *prometheus.CounterVec
}

// helper method to construct the correct "LabelHeaders":"LabelValues" mapping
// to ensure type safety
func getActionSetCounterVecLabels() []kanistermetrics.BoundedLabels {
    bl := make([]kanistermetrics.BoundedLabel, 2)
	  bl[0] = kanistermetrics.BoundedLabel{LabelName: ACTION_SET_COUNTER_VEC_LABEL_RES,
		LabelValues: []string{"success", "failure"}}
    bl[1] = kanistermetrics.BoundedLabel{LabelName:
    ACTION_SET_COUNTER_VEC_LABEL_BLUEPRINT, 
    LabelValues: []string{"backup", "restore"}}
    return bl
}


// helper method to construct the labels correctly. 
func constructActionSetCounterVecLabels(operation_type string, resolution string) prometheus.Labels {
	return prometheus.Labels{ACTION_SET_COUNTER_VEC_LABEL_OP_TYPE: operation_type, 
  ACTION_SET_COUNTER_VEC_LABEL_RES: resolution}
}

// helper method to create a Metrics interface
func newMetrics() *metrics {
    actionSetCounterOpts := prometheus.CounterOpts{
        Name: "action_set_resolutions_total",
        Help: "Total number of action set resolutions",
    }
    actionSetCounterVec := kanistermetrics.InitCounterVec(prometheus.DefaultRegisterer,
		actionSetCounterOpts, getActionSetCounterVecLabels())
    return &metrics{ActionSetCounterVec: actionSetCounterVec}
}
```

The below example will show how the above created ActionSetCounterVec will be incremented in a method:

```golang
func (c *Controller) handleActionSet(ctx context.Context) {
  c.metrics.ActionSetCounterVec.With(constructActionSetCounterVecLabels(  "backup", "success")).Inc()
}
```

Alternatively, one can also directly call the prometheus API with 
positional arguments:

```golang
func (c *Controller) handleActionSet(ctx context.Context) {
  c.metrics.ActionSetCounterVec.WithLabelValues("backup", "success").Inc()
}
```


## Testing

1. The testing will include manual testing of whether the metrics added are successfully getting exported to kanister. 

2. The interfaces listed above in the metrics package will not be unit tested, since they would be testing the behavior of the Prometheus API itself, which breaks the chain of trust principle with dependencies in unit testing.  

3. Integration tests will be added for code that exports new metrics, to ensure that the behavior of exporting metrics is correct. 