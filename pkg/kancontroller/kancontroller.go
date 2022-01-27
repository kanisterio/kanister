/*
Copyright 2020 The Kanister Authors.

Copyright 2016 The Rook Authors. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Some of the code below came from https://github.com/coreos/etcd-operator
which also has the apache 2.0 license.
*/

// Package for a kanister operator
package kancontroller

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"k8s.io/client-go/rest"

	"github.com/kanisterio/kanister/pkg/controller"
	_ "github.com/kanisterio/kanister/pkg/function"
	"github.com/kanisterio/kanister/pkg/handler"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/resource"
)

func Execute() {
	ctx := context.Background()

	// Initialize the clients.
	log.Print("Getting kubernetes context")
	config, err := rest.InClusterConfig()
	if err != nil {
		log.WithError(err).Print("Failed to get k8s config")
		return
	}

	// Make sure the CRD's exist.
	if err := resource.CreateCustomResources(ctx, config); err != nil {
		log.WithError(err).Print("Failed to create CustomResources.")
		return
	}

	ns, err := kube.GetControllerNamespace()
	if err != nil {
		log.WithError(err).Print("Failed to determine this pod's namespace.")
		return
	}

	// Create and start the watcher.
	ctx, cancel := context.WithCancel(ctx)
	c := controller.New(config)
	err = c.StartWatch(ctx, ns)
	if err != nil {
		log.WithError(err).Print("Failed to start controller.")
		cancel()
		return
	}

	err = handler.RunWebhookServer()
	if err != nil {
		log.WithError(err).Print("Failed to start kanister controller server")
	}

	// create signals to stop watching the resources
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	<-signalChan
	log.Print("shutdown signal received, exiting...")
	cancel()
}
