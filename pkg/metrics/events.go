package metrics

// This file contains wrapper functions that will map a Prometheus metric names to its
// label field, help field and an associated Event.

// MetricType will represent a Prometheus metric.
// A variable of type MetricType will hold the name of the Prometheus metric as reported.
type MetricType string

const (
	SampleCountType MetricType = "sample_count"

	ActionSetBackupCreatedType   MetricType = "actionset_backup_created_count"
	ActionSetBackupCompletedType MetricType = "actionset_backup_completed_count"

	ActionSetRestoreCreatedType   MetricType = "actionset_restore_created_count"
	ActionSetRestoreCompletedType MetricType = "actionset_restore_completed_count"

	ActionSetTotalCreatedType   MetricType = "actionset_total_created_count"
	ActionSetTotalCompletedType MetricType = "actionset_total_completed_count"
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
var MetricTypeOpts = map[MetricType]MetricTypeOpt{
	SampleCountType: {
		EventFunc:  NewSampleCount,
		Help:       "Sample counter to remove later",
		LabelNames: []string{"sample"},
	},

	ActionSetBackupCreatedType: {
		EventFunc: NewActionSetBackupCreated,
		Help:      "The count of backup ActionSets created",
	},
	ActionSetBackupCompletedType: {
		EventFunc:  NewActionSetBackupCompleted,
		Help:       "The count of backup ActionSets completed",
		LabelNames: []string{"state"},
	},

	ActionSetRestoreCreatedType: {
		EventFunc: NewActionSetRestoreCreated,
		Help:      "The count of restore ActionSets created",
	},
	ActionSetRestoreCompletedType: {
		EventFunc:  NewActionSetRestoreCompleted,
		Help:       "The count of restore ActionSets completed",
		LabelNames: []string{"state"},
	},

	ActionSetTotalCreatedType: {
		EventFunc:  NewActionSetTotalCreated,
		Help:       "The count of total ActionSets created",
		LabelNames: []string{"actionType", "namespace"},
	},
	ActionSetTotalCompletedType: {
		EventFunc:  NewActionSetTotalCompleted,
		Help:       "The count of total ActionSets completed",
		LabelNames: []string{"actionType", "namespace", "state"},
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

func NewSampleCount(sample string) Event {
	return Event{
		eventType: SampleCountType,
		labels: map[string]string{
			"sample": sample,
		},
	}
}

func NewActionSetBackupCreated() Event {
	return Event{
		eventType: ActionSetBackupCreatedType,
	}
}

func NewActionSetBackupCompleted(state string) Event {
	return Event{
		eventType: ActionSetBackupCompletedType,
		labels: map[string]string{
			"state": state,
		},
	}
}

func NewActionSetRestoreCreated() Event {
	return Event{
		eventType: ActionSetRestoreCreatedType,
	}
}

func NewActionSetRestoreCompleted(state string) Event {
	return Event{
		eventType: ActionSetRestoreCompletedType,
		labels: map[string]string{
			"state": state,
		},
	}
}

func NewActionSetTotalCreated(actionType string, namespace string) Event {
	return Event{
		eventType: ActionSetTotalCreatedType,
		labels: map[string]string{
			"actionType": actionType,
			"namespace":  namespace,
		},
	}
}

func NewActionSetTotalCompleted(actionType string, namespace string, state string) Event {
	return Event{
		eventType: ActionSetTotalCompletedType,
		labels: map[string]string{
			"actionType": actionType,
			"namespace":  namespace,
			"state":      state,
		},
	}
}
