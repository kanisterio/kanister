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
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-logr/logr"
	"github.com/kanisterio/errkit"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/kanisterio/kanister/pkg/validatingwebhook"
	"github.com/kanisterio/kanister/pkg/version"
)

const (
	healthCheckAddr = ":8081"
	livenessPath    = "/healthz"
	readinessPath   = "/readyz"
	metricsPath     = "/metrics"
	whHandlePath    = "/validate/v1alpha1/blueprint"
)

// Info provides information about kanister controller
type Info struct {
	Alive   bool   `json:"alive"`
	Version string `json:"version"`
}

var _ http.Handler = (*healthCheckHandler)(nil)

type healthCheckHandler struct{}

func (*healthCheckHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	version := version.Version
	info := Info{true, version}
	js, err := json.Marshal(info)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.Writer.Write(w, js)
}

func combinedReadinessCheck(mgr ctrl.Manager) healthz.Checker {
	return func(_ *http.Request) error {
		if !mgr.GetCache().WaitForCacheSync(context.Background()) {
			return fmt.Errorf("controller cache not synced")
		}

		// Do we need to check for leader election? as well for reposervercontroller I can see
		// leader election is disabled.
		// Can add more checks if any downstream dependency is there.

		return nil
	}
}

// RunWebhookServer starts the validating webhook resources for blueprint kanister resources
func RunWebhookServer(c *rest.Config) error {
	log.SetLogger(logr.New(log.NullLogSink{}))
	mgr, err := manager.New(c, manager.Options{
		HealthProbeBindAddress: healthCheckAddr,
		LivenessEndpointName:   livenessPath,
		ReadinessEndpointName:  readinessPath,
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

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		return err
	}

	return nil
}

func NewServer() *http.Server {
	m := &http.ServeMux{}
	m.Handle(livenessPath, &healthCheckHandler{})
	m.Handle(metricsPath, promhttp.Handler())
	return &http.Server{Addr: healthCheckAddr, Handler: m}
}
