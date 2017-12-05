package handler

import (
	"io"
	"net/http"
)

const (
	healthCheckPath = "/v0/healthz"
	healthCheckAddr = ":8000"
)

var _ http.Handler = (*healthCheckHandler)(nil)

type healthCheckHandler struct{}

func NewHealthCheckHandler() *healthCheckHandler {
	return &healthCheckHandler{}
}

func (*healthCheckHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"alive": true}`)
}

func NewServer() *http.Server {
	m := &http.ServeMux{}
	m.Handle(healthCheckPath, &healthCheckHandler{})
	return &http.Server{Addr: healthCheckAddr, Handler: m}
}
