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

// This is an internal (package ephemeral) test so it can save/restore the
// unexported podCredentialInjectors registry between cases. It uses the
// standard testing package rather than gocheck, mirroring
// pkg/objectstore/azure_sas_test.go.
package ephemeral

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
)

// resetInjectors clears the global registry for the duration of a test and
// restores it afterwards, so cases don't leak injectors into one another.
func resetInjectors(t *testing.T) {
	t.Helper()
	saved := podCredentialInjectors
	podCredentialInjectors = nil
	t.Cleanup(func() { podCredentialInjectors = saved })
}

// injectorFunc builds a PodCredentialInjectorFunc from a simpler closure that
// only needs the pod options, so individual tests stay terse.
func injectorFunc(fn func(o *kube.PodOptions) (func(context.Context), error)) PodCredentialInjectorFunc {
	return func(_ context.Context, _ kubernetes.Interface, _ param.TemplateParams, _ []string, o *kube.PodOptions) (func(context.Context), error) {
		return fn(o)
	}
}

func TestInjectPodCredentialsNoInjectors(t *testing.T) {
	resetInjectors(t)

	cleanup, err := InjectPodCredentials(context.Background(), nil, param.TemplateParams{}, nil, &kube.PodOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cleanup == nil {
		t.Fatal("cleanup must be non-nil even when no injectors are registered")
	}
	cleanup(context.Background()) // must be a safe no-op
}

func TestInjectPodCredentialsMutatesOptions(t *testing.T) {
	resetInjectors(t)

	RegisterPodCredentialInjector(injectorFunc(func(o *kube.PodOptions) (func(context.Context), error) {
		if o.Labels == nil {
			o.Labels = map[string]string{}
		}
		o.Labels["injected"] = "true"
		return nil, nil // no auxiliary object, so no cleanup
	}))

	opts := &kube.PodOptions{}
	if _, err := InjectPodCredentials(context.Background(), nil, param.TemplateParams{}, nil, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Labels["injected"] != "true" {
		t.Errorf("injector did not mutate options; labels = %v", opts.Labels)
	}
}

func TestInjectPodCredentialsCleanupIsLIFO(t *testing.T) {
	resetInjectors(t)

	var order []string
	mk := func(id string) PodCredentialInjectorFunc {
		return injectorFunc(func(*kube.PodOptions) (func(context.Context), error) {
			return func(context.Context) { order = append(order, id) }, nil
		})
	}
	RegisterPodCredentialInjector(mk("first"))
	RegisterPodCredentialInjector(mk("second"))

	cleanup, err := InjectPodCredentials(context.Background(), nil, param.TemplateParams{}, nil, &kube.PodOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cleanup(context.Background())

	want := []string{"second", "first"}
	if !reflect.DeepEqual(order, want) {
		t.Errorf("cleanup order = %v, want %v (reverse of registration)", order, want)
	}
}

func TestInjectPodCredentialsNilCleanupSkipped(t *testing.T) {
	resetInjectors(t)

	var ran []string
	RegisterPodCredentialInjector(injectorFunc(func(*kube.PodOptions) (func(context.Context), error) {
		return nil, nil // returns a nil cleanup
	}))
	RegisterPodCredentialInjector(injectorFunc(func(*kube.PodOptions) (func(context.Context), error) {
		return func(context.Context) { ran = append(ran, "real") }, nil
	}))

	cleanup, err := InjectPodCredentials(context.Background(), nil, param.TemplateParams{}, nil, &kube.PodOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cleanup(context.Background()) // must not panic on the nil cleanup

	if !reflect.DeepEqual(ran, []string{"real"}) {
		t.Errorf("ran = %v, want [real]", ran)
	}
}

func TestInjectPodCredentialsErrorRollsBack(t *testing.T) {
	resetInjectors(t)

	var cleaned []string
	// First injector succeeds and registers a cleanup.
	RegisterPodCredentialInjector(injectorFunc(func(*kube.PodOptions) (func(context.Context), error) {
		return func(context.Context) { cleaned = append(cleaned, "A") }, nil
	}))
	// Second injector fails; A's cleanup must be invoked before the error returns.
	wantErr := errors.New("mint failed")
	RegisterPodCredentialInjector(injectorFunc(func(*kube.PodOptions) (func(context.Context), error) {
		return nil, wantErr
	}))

	cleanup, err := InjectPodCredentials(context.Background(), nil, param.TemplateParams{}, nil, &kube.PodOptions{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	if !reflect.DeepEqual(cleaned, []string{"A"}) {
		t.Errorf("rollback cleanups = %v, want [A]", cleaned)
	}

	// The returned cleanup must be a no-op: rollback already ran, so calling it
	// must not double-clean.
	cleanup(context.Background())
	if !reflect.DeepEqual(cleaned, []string{"A"}) {
		t.Errorf("returned cleanup should be a no-op after rollback; cleaned = %v", cleaned)
	}
}
