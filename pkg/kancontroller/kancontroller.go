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

// Package kancontroller is for a kanister operator.
package kancontroller

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/rest"

	"github.com/kanisterio/kanister/pkg/controller"
	"github.com/kanisterio/kanister/pkg/handler"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/resource"
	"github.com/kanisterio/kanister/pkg/validatingwebhook"

	_ "github.com/kanisterio/kanister/pkg/function"
)

const (
	kanisterMetricsEnv = "KANISTER_METRICS_ENABLED"
)

// metricsEnabled checks if the feature flag for kanister metrics is enabled
// If the environment variable is not set, then it returns a default
// "false" value.
func metricsEnabled() bool {
	metricsEnabled, ok := os.LookupEnv(kanisterMetricsEnv)
	if !ok {
		log.Error().Print("KANISTER_METRICS_ENABLED env variable not set")
		return false
	}
	enabled, err := strconv.ParseBool(metricsEnabled)
	if err != nil {
		log.Error().Print("Error parsing KANISTER_METRICS_ENABLED env variable to bool")
		return false
	}
	return enabled
}

func Execute() {
	ctx := context.Background()
	logLevel, exists := os.LookupEnv(log.LevelEnvName)
	if exists {
		log.Print(fmt.Sprintf("Controller log level: %s", logLevel))
	}
	// Initialize the clients.
	log.Print("Getting kubernetes context")
	config, err := rest.InClusterConfig()
	if err != nil {
		log.WithError(err).Print("Failed to get k8s config")
		return
	}

	// Run HTTPS webhook server if webhook certificates are mounted in the pod
	// otherwise normal HTTP server for health and prom endpoints
	if validatingwebhook.IsCACertMounted() {
		go func(config *rest.Config) {
			err := handler.RunWebhookServer(config)
			if err != nil {
				log.WithError(err).Print("Failed to start validating webhook server")
				return
			}
		}(config)
	} else {
		s := handler.NewServer()
		defer func() {
			if err := s.Shutdown(ctx); err != nil {
				log.WithError(err).Print("Failed to shutdown health check server")
			}
		}()
		go func() {
			if err := s.ListenAndServe(); err != nil {
				log.WithError(err).Print("Failed to start health check server")
			}
		}()
	}

	// CRDs should only be created/updated if the env var CREATEORUPDATE_CRDS is set to true
	if resource.CreateOrUpdateCRDs() {
		if err := resource.CreateCustomResources(ctx, config); err != nil {
			log.WithError(err).Print("Failed to create CustomResources.")
			return
		}
	}

	ns, err := kube.GetControllerNamespace()
	if err != nil {
		log.WithError(err).Print("Failed to determine this pod's namespace.")
		return
	}

	// Create and start the watcher.
	ctx, cancel := context.WithCancel(ctx)

	var c *controller.Controller

	// pass a new prometheus registry or nil depending on
	// the kanister prometheus metrics feature flag
	if metricsEnabled() {
		c = controller.New(config, prometheus.DefaultRegisterer)
	} else {
		c = controller.New(config, nil)
	}
	err = c.StartWatch(ctx, ns)
	if err != nil {
		log.WithError(err).Print("Failed to start controller.")
		cancel()
		return
	}

	// create signals to stop watching the resources
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	<-signalChan
	log.Print("shutdown signal received, exiting...")
	cancel()
}
