package handler

import (
	"encoding/json"
	"io"
	"net/http"

	kanister "github.com/kanisterio/kanister/pkg"
)

const (
	healthCheckPath = "/v0/healthz"
	healthCheckAddr = ":8000"
)

type Info struct {
	Alive   bool   `json:"alive"`
	Version string `json:"version"`
}

var _ http.Handler = (*healthCheckHandler)(nil)

type healthCheckHandler struct{}

func NewHealthCheckHandler() *healthCheckHandler {
	return &healthCheckHandler{}
}

func (*healthCheckHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	version := kanister.VERSION
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

func NewServer() *http.Server {
	m := &http.ServeMux{}
	m.Handle(healthCheckPath, &healthCheckHandler{})
	return &http.Server{Addr: healthCheckAddr, Handler: m}
}
