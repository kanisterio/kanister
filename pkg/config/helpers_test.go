/*
Copyright 2026 by contributors to the Kanister project.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"gopkg.in/check.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

const serviceNameEnvName = "SERVICE_NAME"

type HelpersSuite struct {
	savedEnv map[string]*string
}

var _ = check.Suite(&HelpersSuite{})

func Test(t *testing.T) {
	check.TestingT(t)
}

func (s *HelpersSuite) SetUpTest(c *check.C) {
	s.savedEnv = map[string]*string{}
	for _, name := range []string{ClusterNameEnvName, serviceNameEnvName} {
		if v, ok := os.LookupEnv(name); ok {
			s.savedEnv[name] = &v
		} else {
			s.savedEnv[name] = nil
		}
		c.Assert(os.Unsetenv(name), check.IsNil)
	}
}

func (s *HelpersSuite) TearDownTest(c *check.C) {
	for name, v := range s.savedEnv {
		if v == nil {
			c.Assert(os.Unsetenv(name), check.IsNil)
		} else {
			c.Assert(os.Setenv(name, *v), check.IsNil)
		}
	}
}

func newFakeClientWithDefaultNamespace(uid string) kubernetes.Interface {
	return fake.NewClientset(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultNamespaceName,
			UID:  types.UID(uid),
		},
	})
}

func (s *HelpersSuite) TestGetClusterNameFromEnv(c *check.C) {
	c.Assert(os.Setenv(ClusterNameEnvName, "my-cluster"), check.IsNil)

	name, err := GetClusterName(fake.NewClientset())
	c.Assert(err, check.IsNil)
	c.Assert(name, check.Equals, "my-cluster")
}

func (s *HelpersSuite) TestGetClusterNameFromDefaultNamespaceUID(c *check.C) {
	name, err := GetClusterName(newFakeClientWithDefaultNamespace("default-ns-uid"))
	c.Assert(err, check.IsNil)
	c.Assert(name, check.Equals, "default-ns-uid")
}

func (s *HelpersSuite) TestGetClusterNameIgnoresServiceName(c *check.C) {
	c.Assert(os.Setenv(serviceNameEnvName, "my-service"), check.IsNil)

	name, err := GetClusterName(newFakeClientWithDefaultNamespace("default-ns-uid"))
	c.Assert(err, check.IsNil)
	c.Assert(name, check.Equals, "default-ns-uid")
}

func (s *HelpersSuite) TestGetClusterNameMissingDefaultNamespace(c *check.C) {
	name, err := GetClusterName(fake.NewClientset())
	c.Assert(err, check.NotNil)
	c.Assert(name, check.Equals, "")
}

func (s *HelpersSuite) TestGetClusterNameTimesOut(c *check.C) {
	// API server which never responds until the request is canceled.
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer srv.Close()
	defer close(release)

	cli, err := kubernetes.NewForConfig(&rest.Config{Host: srv.URL})
	c.Assert(err, check.IsNil)

	savedTimeout := clusterNameLookupTimeout
	clusterNameLookupTimeout = 100 * time.Millisecond
	defer func() { clusterNameLookupTimeout = savedTimeout }()

	done := make(chan error, 1)
	go func() {
		_, err := GetClusterName(cli)
		done <- err
	}()

	select {
	case err := <-done:
		c.Assert(errors.Is(err, context.DeadlineExceeded), check.Equals, true, check.Commentf("unexpected error: %v", err))
	case <-time.After(10 * time.Second):
		c.Fatal("GetClusterName did not time out")
	}
}
