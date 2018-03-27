package eventer

import (
	log "github.com/sirupsen/logrus"
	core "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
)

//NewEventRecorder returns an EventRecorder to records events for the associated runtime object
func NewEventRecorder(client kubernetes.Interface, component string) record.EventRecorder {
	// Event Broadcaster
	broadcaster := record.NewBroadcaster()
	broadcaster.StartEventWatcher(
		func(event *core.Event) {
			if _, err := client.Core().Events(event.Namespace).Create(event); err != nil {
				log.Errorf("Error while creating the event: %#v", err)
			}
		},
	)

	return broadcaster.NewRecorder(scheme.Scheme, core.EventSource{Component: component})
}
