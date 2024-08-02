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

package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/kanisterio/kanister/pkg/validatingwebhook"
	"github.com/kanisterio/kanister/pkg/version"
)

const (
	healthCheckPath = "/v0/healthz"
	metricsPath     = "/metrics"
	healthCheckAddr = ":8000"
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
	_, _ = io.WriteString(w, string(js))
}

// RunWebhookServer starts the validating webhook resources for blueprint kanister resources
func RunWebhookServer(c *rest.Config) error {
	log.SetLogger(logr.New(log.NullLogSink{}))
	mgr, err := manager.New(c, manager.Options{})
	if err != nil {
		return errors.Wrap(err, "Failed to create new webhook manager")
	}
	bpValidator := &validatingwebhook.BlueprintValidator{}
	if err = bpValidator.InjectDecoder(admission.NewDecoder(mgr.GetScheme())); err != nil {
		return errors.Wrap(err, "Failed to inject decoder")
	}

	hookServerOptions := webhook.Options{CertDir: validatingwebhook.WHCertsDir}
	hookServer := webhook.NewServer(hookServerOptions)
	hookServer.Register(whHandlePath, &webhook.Admission{Handler: bpValidator})
	hookServer.Register(healthCheckPath, &healthCheckHandler{})
	hookServer.Register(metricsPath, promhttp.Handler())

	if err := mgr.Add(hookServer); err != nil {
		return errors.Wrap(err, "Failed to add new webhook server")
	}

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		return err
	}

	return nil
}

func NewServer() *http.Server {
	m := &http.ServeMux{}
	m.Handle(healthCheckPath, &healthCheckHandler{})
	m.Handle(metricsPath, promhttp.Handler())
	return &http.Server{Addr: healthCheckAddr, Handler: m}
}
