package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/kanisterio/kanister/pkg/version"
)

const (
	healthCheckPath = "/v0/healthz"
	healthCheckAddr = ":8000"
)

// Info provides information about kanister controller
type Info struct {
	Alive   bool   `json:"alive"`
	Version string `json:"version"`
}

var _ http.Handler = (*healthCheckHandler)(nil)

type healthCheckHandler struct{}

//NewHealthCheckHandler function returns pointer to an empty healthCheckHandler
func NewHealthCheckHandler() *healthCheckHandler {
	return &healthCheckHandler{}
}

func (*healthCheckHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	version := version.VERSION
	info := Info{true, version}
	js, err := json.Marshal(info)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, string(js))
}

// NewServer returns a pointer to the http Server
func NewServer() *http.Server {
	m := &http.ServeMux{}
	m.Handle(healthCheckPath, &healthCheckHandler{})
	return &http.Server{Addr: healthCheckAddr, Handler: m}
}
