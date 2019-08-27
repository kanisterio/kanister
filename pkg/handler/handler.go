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
