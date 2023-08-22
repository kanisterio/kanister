// Copyright 2023 The Kanister Authors.
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

package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	crkanisteriov1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/controllers/repositoryserver"
	"github.com/kanisterio/kanister/pkg/handler"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/resource"
	"github.com/kanisterio/kanister/pkg/validatingwebhook"
	//+kubebuilder:scaffold:imports
)

var (
	scheme          = runtime.NewScheme()
	setupLog        = ctrl.Log.WithName("setup")
	defaultLogLevel = zapcore.InfoLevel
)

const (
	WHCertsDir        = "/var/run/webhook/serving-cert"
	whHandlePath      = "/validate/v1alpha1/repositoryserver"
	webhookServerPort = 8443
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(crkanisteriov1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var probeAddr string
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	logLevel := getLogLevel()

	opts := zap.Options{
		Level: logLevel,
	}

	// Disable metrics server
	metricsAddr = "0"
	opts.BindFlags(flag.CommandLine)
	flag.Parse()
	logger := zap.New(zap.UseFlagOptions(&opts))
	ctrl.SetLogger(logger)
	config := ctrl.GetConfigOrDie()
	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         false,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&repositoryserver.RepositoryServerReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RepositoryServer")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	// CRDs should only be created/updated if the env var CREATEORUPDATE_CRDS is set to true
	if resource.CreateOrUpdateCRDs() {
		if err := resource.CreateRepoServerCustomResource(context.Background(), config); err != nil {
			setupLog.Error(err, "Failed to create CustomResources.")
			os.Exit(1)
		}
	}

	if isCACertMounted() {
		hookServer := mgr.GetWebhookServer()
		webhook := admission.WithCustomValidator(&v1alpha1.RepositoryServer{}, &validatingwebhook.RepositoryServerWebhook{})
		hookServer.Register(whHandlePath, webhook)
		hookServer.CertDir = WHCertsDir
		hookServer.Port = webhookServerPort
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func getLogLevel() zapcore.Level {
	logLevel := os.Getenv(log.LevelEnvName)
	if logLevel == "" {
		return defaultLogLevel
	}
	level, err := zapcore.ParseLevel(logLevel)
	if err != nil {
		return defaultLogLevel
	}
	return level
}

func isCACertMounted() bool {
	if _, err := os.Stat(fmt.Sprintf("%s/%s", handler.WHCertsDir, "tls.crt")); err != nil {
		return false
	}

	return true
}
