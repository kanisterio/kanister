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

package testing

import (
	"os"
	"testing"

	. "gopkg.in/check.v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/kube"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {
	integrationSetup(t)
	TestingT(t)
	integrationCleanup(t)
}

const (
	controllerSA = "default"
)

// SetupIntegration just creates the controller namespace
func integrationSetup(t *testing.T) {
	cfg, err := kube.LoadConfig()
	if err != nil {
		t.Fatalf("Integration test setup failure: Error loading kube.Config; err=%v", err)
	}
	cli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		t.Fatalf("Integration test setup failure: Error createing kubeCli; err=%v", err)
	}
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: consts.ControllerNS,
		},
	}
	if _, err = cli.CoreV1().Namespaces().Create(ns); err != nil {
		t.Fatalf("Integration test setup failure: Error createing namespace; err=%v", err)
	}

	//  Set Controller namespace and service account
	os.Setenv(kube.PodNSEnvVar, consts.ControllerNS)
	os.Setenv(kube.PodSAEnvVar, controllerSA)
}

func integrationCleanup(t *testing.T) {
	cfg, err := kube.LoadConfig()
	if err != nil {
		t.Fatalf("Integration test cleanup failure: Error loading kube.Config; err=%v", err)
	}
	cli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		t.Fatalf("Integration test cleanup failure: Error createing kubeCli; err=%v", err)
	}
	if err := cli.CoreV1().Namespaces().Delete(consts.ControllerNS, &metav1.DeleteOptions{}); err != nil {
		t.Fatalf("Integration test cleanup failure: Error deleting namespace; err=%v", err)
	}
}
