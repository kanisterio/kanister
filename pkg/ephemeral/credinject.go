// Copyright 2026 The Kanister Authors.
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

package ephemeral

import (
	"context"

	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
)

// PodCredentialInjector augments a to-be-created ephemeral pod with credential
// material that must be minted or created at pod-build time — for example a
// short-lived cloud SAS token delivered via an ephemeral Secret and a projected
// file — rather than the static, in-memory mutations performed by an Applier.
//
// Unlike an Applier (whose Apply only receives *PodOptions), an injector runs
// with a live context, a Kubernetes client and the operation's TemplateParams,
// and may create auxiliary API objects. It therefore returns a cleanup function
// that the caller MUST invoke once the pod has completed, to tear those objects
// down. An injector that does not apply to the current pod should return a nil
// cleanup and nil error.
type PodCredentialInjector interface {
	Inject(
		ctx context.Context,
		cli kubernetes.Interface,
		tp param.TemplateParams,
		command []string,
		options *kube.PodOptions,
	) (cleanup func(context.Context), err error)
}

// PodCredentialInjectorFunc adapts an ordinary function to the
// PodCredentialInjector interface.
type PodCredentialInjectorFunc func(
	context.Context,
	kubernetes.Interface,
	param.TemplateParams,
	[]string,
	*kube.PodOptions,
) (func(context.Context), error)

func (f PodCredentialInjectorFunc) Inject(
	ctx context.Context,
	cli kubernetes.Interface,
	tp param.TemplateParams,
	command []string,
	options *kube.PodOptions,
) (func(context.Context), error) {
	return f(ctx, cli, tp, command, options)
}

// podCredentialInjectors is the process-wide registry of injectors, populated
// at init time by consumers (e.g. K10 registers an Azure Workload Identity SAS
// injector).
var podCredentialInjectors []PodCredentialInjector

// RegisterPodCredentialInjector adds an injector to the registry. It is not
// safe for concurrent use and is expected to be called from init().
func RegisterPodCredentialInjector(injector PodCredentialInjector) {
	podCredentialInjectors = append(podCredentialInjectors, injector)
}

// InjectPodCredentials runs every registered injector against the pod options.
// It returns a single cleanup function that reverses the injectors in LIFO
// order; the caller must defer it until after the pod has completed. If any
// injector fails, the injectors that already ran are cleaned up before the
// error is returned, and the returned cleanup is a no-op.
func InjectPodCredentials(
	ctx context.Context,
	cli kubernetes.Interface,
	tp param.TemplateParams,
	command []string,
	options *kube.PodOptions,
) (func(context.Context), error) {
	var cleanups []func(context.Context)
	runCleanups := func(ctx context.Context) {
		for i := len(cleanups) - 1; i >= 0; i-- {
			if cleanups[i] != nil {
				cleanups[i](ctx)
			}
		}
	}

	for _, injector := range podCredentialInjectors {
		cleanup, err := injector.Inject(ctx, cli, tp, command, options)
		if err != nil {
			runCleanups(ctx)
			return func(context.Context) {}, err
		}
		if cleanup != nil {
			cleanups = append(cleanups, cleanup)
		}
	}

	return runCleanups, nil
}
