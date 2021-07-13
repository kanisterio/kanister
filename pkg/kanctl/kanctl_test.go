// Copyright 2020 The Kanister Authors.
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

package kanctl

import (
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned"
	osversioned "github.com/openshift/client-go/apps/clientset/versioned"
	. "gopkg.in/check.v1"
	"k8s.io/client-go/kubernetes"
)

func (k *KanctlTestSuite) TestInitializeClients(c *C) {
	clients, err := initializeClients()

	// No errors are thrown
	c.Assert(err, IsNil)

	// return value has the correct type
	c.Assert(clients, FitsTypeOf, &Clients{})

	// check all struct fields
	var kubeInterface kubernetes.Interface

	c.Assert(clients.KubeClient, Implements, &kubeInterface)

	var verInterface versioned.Interface

	c.Assert(clients.CrdClient, Implements, &verInterface)

	var osVerInterface osversioned.Interface

	c.Assert(clients.OsClient, Implements, &osVerInterface)
}
