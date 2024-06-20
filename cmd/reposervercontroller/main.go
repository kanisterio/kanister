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
	"os"
	"strconv"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/discovery"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/controllers/repositoryserver"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/resource"
	"github.com/kanisterio/kanister/pkg/validatingwebhook"
	//nolint:gci
	//+kubebuilder:scaffold:imports

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var (
	scheme          = runtime.NewScheme()
	setupLog        = ctrl.Log.WithName("setup")
	defaultLogLevel = zapcore.InfoLevel
)

const (
	whHandlePath      = "/validate/v1alpha1/repositoryserver"
	webhookServerPort = 8443
	minorK8sVersion   = 25
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(crv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var probeAddr string
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	logLevel := getLogLevel()

	log.SetupClusterNameInLogVars()

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
		Metrics:                server.Options{BindAddress: metricsAddr},
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

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		setupLog.Error(err, "Failed to get discovery client")
		os.Exit(1)
	}

	k8sserverVersion, err := discoveryClient.ServerVersion()
	if err != nil {
		setupLog.Error(err, "Failed to get server version using discovery client")
		os.Exit(1)
	}

	minorVersion, err := strconv.Atoi(k8sserverVersion.Minor)
	if err != nil {
		setupLog.Error(err, "Failed to convert to integer")
		os.Exit(1)
	}

	// We are using CEL validation rules for k8s server version > 1.25.
	// More information about CEL can be found here - https://kubernetes.io/blog/2022/09/23/crd-validation-rules-beta/
	// CEL is not supported for k8s server versions below 1.25. Hence for backward compatibility
	// we can use validating webhook for k8s server versions < 1.25
	if k8sserverVersion.Major == "1" && minorVersion < minorK8sVersion {
		if validatingwebhook.IsCACertMounted() {
			hookServerOptions := webhook.Options{CertDir: validatingwebhook.WHCertsDir, Port: webhookServerPort}
			hookServer := webhook.NewServer(hookServerOptions)
			webhook := admission.WithCustomValidator(mgr.GetScheme(), &crv1alpha1.RepositoryServer{}, &validatingwebhook.RepositoryServerValidator{})
			// registers a webhooks to a webhook server that gets run by a controller manager.
			hookServer.Register(whHandlePath, webhook)
			if err := mgr.Add(hookServer); err != nil {
				setupLog.Error(err, "Failed to add webhook server to the manager")
				os.Exit(1)
			}
		}
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
