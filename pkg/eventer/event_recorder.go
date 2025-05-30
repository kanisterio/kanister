// Copyright 2019 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package eventer provides utilities for recording Kubernetes events.
package eventer

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"

	"github.com/kanisterio/kanister/pkg/log"
)

// NewEventRecorder returns an EventRecorder to records events for the associated runtime object
func NewEventRecorder(client kubernetes.Interface, component string) record.EventRecorder {
	// Event Broadcaster
	broadcaster := record.NewBroadcaster()
	broadcaster.StartEventWatcher(
		func(event *corev1.Event) {
			if _, err := client.CoreV1().Events(event.Namespace).Create(context.TODO(), event, metav1.CreateOptions{}); err != nil {
				log.Error().WithError(err).Print("Error while creating the event")
			}
		},
	)

	return broadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: component})
}
