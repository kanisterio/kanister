package metrics

// This file contains wrapper functions that will map a Prometheus metric names to its
// label field, help field and an associated Event.

// MetricType will represent a Prometheus metric.
// A variable of type MetricType will hold the name of the Prometheus metric as reported.
type MetricType string

const (
	ActionSetCreatedTotalType   MetricType = "kanister_actionset_created_total"
	ActionSetCompletedTotalType MetricType = "kanister_actionset_completed_total"
	ActionSetFailedTotalType    MetricType = "kanister_actionset_failed_total"
)

// MetricTypeOpt is a struct for a Prometheus metric.
// Help and LabelNames are passed directly to the Prometheus predefined functions.
// EventFunc holds the constructor of the linked Event of a given MetricType.
type MetricTypeOpt struct {
	EventFunc  interface{}
	Help       string
	LabelNames []string
}

// Mapping a Prometheus MetricType to the metric MetricTypeOpt struct.
// Basically, a metric name is mapped to its associated Help and LabelName fields.
// The linked event function (EventFunc) is also mapped to this metric name as a part of MetricTypeOpt.
var MetricCounterOpts = map[MetricType]MetricTypeOpt{
	ActionSetCreatedTotalType: {
		EventFunc:  NewActionSetCreatedTotal,
		Help:       "The count of total ActionSets created",
		LabelNames: []string{"actionType", "namespace"},
	},
	ActionSetCompletedTotalType: {
		EventFunc:  NewActionSetCompletedTotal,
		Help:       "The count of total ActionSets completed",
		LabelNames: []string{"actionName", "actionType", "blueprint", "namespace", "state"},
	},
	ActionSetFailedTotalType: {
		EventFunc:  NewActionSetFailedTotal,
		Help:       "The count of total ActionSets failed",
		LabelNames: []string{"actionName", "actionType", "blueprint", "namespace", "state"},
	},
}

// Event describes an individual event.
// eventType is the MetricType with which the individial Event is associated.
// Labels are the metric labels that will be passed to Prometheus.
//
// Note: The type and labels are private in order to force the use of the
// event constructors below. This helps to prevent an event from being
// accidentally misconstructed (e.g. with mismatching labels), which would
// cause the Prometheus library to panic.
type Event struct {
	eventType MetricType
	labels    map[string]string
}

// MetricType returns the event's type.
func (e *Event) Type() MetricType {
	return e.eventType
}

// Labels returns a copy of the event's labels.
func (e *Event) Labels() map[string]string {
	labels := make(map[string]string)
	for k, v := range e.labels {
		labels[k] = v
	}
	return labels
}

func NewActionSetCreatedTotal(actionType string, namespace string) Event {
	return Event{
		eventType: ActionSetCreatedTotalType,
		labels: map[string]string{
			"actionType": actionType,
			"namespace":  namespace,
		},
	}
}

func NewActionSetCompletedTotal(actionName string, actionType string, blueprint string, namespace string, state string) Event {
	return Event{
		eventType: ActionSetCompletedTotalType,
		labels: map[string]string{
			"actionName": actionName,
			"actionType": actionType,
			"blueprint":  blueprint,
			"namespace":  namespace,
			"state":      state,
		},
	}
}

func NewActionSetFailedTotal(actionName string, actionType string, blueprint string, namespace string, state string) Event {
	return Event{
		eventType: ActionSetFailedTotalType,
		labels: map[string]string{
			"actionName": actionName,
			"actionType": actionType,
			"blueprint":  blueprint,
			"namespace":  namespace,
			"state":      state,
		},
	}
}
