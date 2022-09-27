// Copyright 2022 The Kanister Authors.
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

package repositoryserver

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/handler"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
)

const (
	createOrUpdateCRDEnvVar = "CREATEORUPDATE_CRDS"
)

func StartController() {
	ctx := context.Background()
	logLevel, exists := os.LookupEnv(log.LevelEnvName)
	if exists {
		log.Print(fmt.Sprintf("Controller log level: %s", logLevel))
	}
	// Initialize the clients.
	log.Print("Getting kubernetes context")
	config, err := ctrl.GetConfig()
	if err != nil {
		log.WithError(err).Print("Failed to get k8s config")
		return
	}

	// Run HTTPS webhook server if webhook certificates are mounted in the pod
	// otherwise normal HTTP server for health and prom endpoints
	if isCACertMounted() {
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
	if createOrUpdateCRDs() {
		if err := CreateRepositoryServerResource(config); err != nil {
			log.WithError(err).Print("Failed to create RepositoryServerResource.")
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
	c := NewController(config)
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

func isCACertMounted() bool {
	if _, err := os.Stat(fmt.Sprintf("%s/%s", handler.WHCertsDir, "tls.crt")); err != nil {
		return false
	}

	return true
}

func createOrUpdateCRDs() bool {
	createOrUpdateCRD := os.Getenv(createOrUpdateCRDEnvVar)
	if createOrUpdateCRD == "" {
		return true
	}

	c, err := strconv.ParseBool(createOrUpdateCRD)
	if err != nil {
		log.Print("environment variable", field.M{"CREATEORUPDATE_CRDS": createOrUpdateCRD})
		return true
	}

	return c
}
