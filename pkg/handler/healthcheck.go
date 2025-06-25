package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	"github.com/kanisterio/kanister/pkg/version"
)

const (
	livenessPathEnvVar    = "LIVENESS_PATH"
	readinessPathEnvVar   = "READINESS_PATH"
	healthCheckPortEnvVar = "HEALTH_CHECK_PORT"

	// Default values for health checks.
	defaultLivenessPath    = "/v0/healthz"
	defaultReadinessPath   = "/v0/readyz"
	defaultHealthCheckAddr = "8000"
)

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

type readinessCheckHandler struct {
	mgr ctrl.Manager
}

func (rch *readinessCheckHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := combinedReadinessCheck(rch.mgr)(r); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, `{"status":"ready"}`)
}

func combinedReadinessCheck(mgr ctrl.Manager) healthz.Checker {
	return func(_ *http.Request) error {
		if mgr != nil && !mgr.GetCache().WaitForCacheSync(context.Background()) {
			return fmt.Errorf("controller cache not synced")
		}

		// Do we need to check for leader election? as well for reposervercontroller I can see
		// leader election is disabled.
		// Can add more checks if any downstream dependency is there.

		return nil
	}
}

func getLivenessPath() string {
	if lp := os.Getenv(livenessPathEnvVar); lp != "" {
		return lp
	}

	return defaultLivenessPath
}

func getReadinessPath() string {
	if rp := os.Getenv(readinessPathEnvVar); rp != "" {
		return rp
	}

	return defaultReadinessPath
}

func getHealthAddr() string {
	if hp := os.Getenv(healthCheckPortEnvVar); hp != "" {
		return hp
	}

	return defaultHealthCheckAddr
}
