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
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

type PodOutputMap map[string]interface{}

type PodRunnerFunc func(context.Context, PodController) (PodOutputMap, error)

// PodRunner allows us to start / stop pod, write file to pod and execute command within it
type PodRunner interface {
	Run(ctx context.Context, fn func(context.Context, *v1.Pod) (map[string]interface{}, error)) (PodOutputMap, error)
	// RunEx utilizes the PodController interface and forwards it to the functor, simplifying the manipulation with
	// particular pod from the functor.
	// TODO: Since significant number of functions are currently using PodRunner, we'll keep Run for now.
	// However, once all these functions have been refactored to use PodController,
	// Run should be removed and RunEx has to be renamed to Run.
	RunEx(ctx context.Context, fn PodRunnerFunc) (PodOutputMap, error)
}

// PodRunner specifies Kubernetes Client and PodOptions needed for creating Pod
type podRunner struct {
	pc PodController
}

// NewPodRunner returns a new PodRunner given Kubernetes Client and PodOptions
func NewPodRunner(cli kubernetes.Interface, options *PodOptions) PodRunner {
	return &podRunner{
		pc: NewPodController(cli, options),
	}
}

// Run will create a new Pod based on PodRunner contents and execute the given function
func (p *podRunner) Run(ctx context.Context, fn func(context.Context, *v1.Pod) (map[string]interface{}, error)) (PodOutputMap, error) {
	return p.RunEx(ctx, func(innerCtx context.Context, pc PodController) (PodOutputMap, error) {
		return fn(innerCtx, pc.Pod())
	})
}

// RunEx will create a new Pod based on PodRunner contents and execute the given function
func (p *podRunner) RunEx(ctx context.Context, fn PodRunnerFunc) (PodOutputMap, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	err := p.pc.StartPod(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create pod")
	}

	pod := p.pc.Pod()
	ctx = field.Context(ctx, consts.PodNameKey, pod.Name)
	ctx = field.Context(ctx, consts.ContainerNameKey, pod.Spec.Containers[0].Name)
	go func() {
		<-ctx.Done()
		err := p.pc.StopPod(context.Background(), PodControllerInfiniteStopTime, int64(0))
		if err != nil {
			log.WithError(err).Print("Failed to delete pod", field.M{"PodName": pod.Name})
		}
	}()
	return fn(ctx, p.pc)
}
