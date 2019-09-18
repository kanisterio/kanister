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

package kube

import (
	"context"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/field"
)

// PodRunner specifies Kubernetes Client and PodOptions needed for creating Pod
type PodRunner struct {
	cli        kubernetes.Interface
	podOptions *PodOptions
}

// NewPodRunner returns a new PodRunner given Kubernetes Client and PodOptions
func NewPodRunner(cli kubernetes.Interface, options *PodOptions) *PodRunner {
	return &PodRunner{
		cli:        cli,
		podOptions: options,
	}
}

// Run will create a new Pod based on PodRunner contents and execute the given function
func (p *PodRunner) Run(ctx context.Context, fn func(context.Context, *v1.Pod) (map[string]interface{}, error)) (map[string]interface{}, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if p.cli == nil || p.podOptions == nil {
		return nil, errors.New("Pod Runner not initialized")
	}
	pod, err := CreatePod(ctx, p.cli, p.podOptions)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create pod")
	}
	ctx = field.Context(ctx, consts.PodNameKey, pod.Name)
	ctx = field.Context(ctx, consts.ContainerNameKey, pod.Spec.Containers[0].Name)
	go func() {
		<-ctx.Done()
		err := DeletePod(context.Background(), p.cli, pod)
		if err != nil {
			log.Error("Failed to delete pod ", err.Error())
		}
	}()
	return fn(ctx, pod)
}
