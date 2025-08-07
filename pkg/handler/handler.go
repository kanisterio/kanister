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

// Package handler provides functionality for managing HTTP handlers and webhook servers
// used in the Kanister project.
package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	"github.com/kanisterio/errkit"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/kanisterio/kanister/pkg/validatingwebhook"
)

const (
	metricsPath  = "/metrics"
	whHandlePath = "/validate/v1alpha1/blueprint"
)

// Info provides information about kanister controller
type Info struct {
	Alive   bool   `json:"alive"`
	Version string `json:"version"`
}

// RunWebhookServer starts the validating webhook resources for blueprint kanister resources
func RunWebhookServer(ctx context.Context, c *rest.Config) error {
	log.SetLogger(logr.New(log.NullLogSink{}))
	mgr, err := manager.New(c, manager.Options{
		HealthProbeBindAddress: fmt.Sprintf(":%s", getHealthAddr()),
		LivenessEndpointName:   getLivenessPath(),
		ReadinessEndpointName:  getReadinessPath(),
	})
	if err != nil {
		return errkit.Wrap(err, "Failed to create new webhook manager")
	}

	// Register liveness probe.
	// This will always return true, unless the container is down.
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return errkit.Wrap(err, "Failed to add health check")
	}

	// Register readiness probe.
	if err := mgr.AddReadyzCheck("readyz", combinedReadinessCheck(mgr)); err != nil {
		return errkit.Wrap(err, "Failed to add readiness check")
	}

	bpValidator := &validatingwebhook.BlueprintValidator{}
	decoder := admission.NewDecoder(mgr.GetScheme())
	if err = bpValidator.InjectDecoder(&decoder); err != nil {
		return errkit.Wrap(err, "Failed to inject decoder")
	}

	hookServerOptions := webhook.Options{CertDir: validatingwebhook.WHCertsDir}
	hookServer := webhook.NewServer(hookServerOptions)
	hookServer.Register(whHandlePath, &webhook.Admission{Handler: bpValidator})
	hookServer.Register(metricsPath, promhttp.Handler())

	if err := mgr.Add(hookServer); err != nil {
		return errkit.Wrap(err, "Failed to add new webhook server")
	}

	if err := mgr.Start(ctx); err != nil {
		return err
	}

	return nil
}

func NewServer() *http.Server {
	m := &http.ServeMux{}
	m.Handle(getLivenessPath(), &healthCheckHandler{})
	m.Handle(getReadinessPath(), &readinessCheckHandler{})
	m.Handle(metricsPath, promhttp.Handler())
	return &http.Server{Addr: fmt.Sprintf(":%s", getHealthAddr()), Handler: m}
}
