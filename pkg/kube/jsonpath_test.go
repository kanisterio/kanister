// Copyright 2021 The Kanister Authors.
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
	. "gopkg.in/check.v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

type JsonpathSuite struct{}

var _ = Suite(&JsonpathSuite{})

func runtimeObjFromYAML(c *C, specs string) runtime.Object {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(specs), nil, nil)
	c.Assert(err, IsNil)
	return obj
}

func (js *JsonpathSuite) TestDeploymentReady(c *C) {
	deploy := `apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    deployment.kubernetes.io/revision: "1"
  creationTimestamp: "2021-08-30T14:43:29Z"
  generation: 1
  name: test-deployment
  namespace: test
  resourceVersion: "2393578"
  uid: 13b876a9-440f-45ba-8e5f-fea8167b5dc9
spec:
  progressDeadlineSeconds: 600
  replicas: 3
  revisionHistoryLimit: 10
  selector:
    matchLabels:
      app: demo
  strategy:
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
    type: RollingUpdate
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: demo
    spec:
      containers:
      - image: nginx:1.12
        imagePullPolicy: IfNotPresent
        name: web
        ports:
        - containerPort: 80
          name: http
          protocol: TCP
        resources: {}
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      terminationGracePeriodSeconds: 30
status:
  availableReplicas: 3
  conditions:
  - lastTransitionTime: "2021-08-30T14:43:31Z"
    lastUpdateTime: "2021-08-30T14:43:31Z"
    message: Deployment has minimum availability.
    reason: MinimumReplicasAvailable
    status: "True"
    type: Available
  - lastTransitionTime: "2021-08-30T14:43:29Z"
    lastUpdateTime: "2021-08-30T14:43:31Z"
    message: ReplicaSet "test-deployment-6b4d4fbcdb" has successfully progressed.
    reason: NewReplicaSetAvailable
    status: "True"
    type: Progressing
  observedGeneration: 1
  readyReplicas: 3
  replicas: 3
  updatedReplicas: 3
`
	obj := runtimeObjFromYAML(c, deploy)
	replica, err := ResolveJsonpathToString(obj, "{.spec.replicas}")
	c.Assert(err, IsNil)
	c.Assert(replica, Equals, "3")

	readyReplicas, err := ResolveJsonpathToString(obj, "{.status.replicas}")
	c.Assert(err, IsNil)
	c.Assert(readyReplicas, Equals, "3")

	availReplicas, err := ResolveJsonpathToString(obj, "{.status.availableReplicas}")
	c.Assert(err, IsNil)
	c.Assert(availReplicas, Equals, "3")

	// Any condition with type Available
	condType, err := ResolveJsonpathToString(obj, `{.status.conditions[?(@.type == "Available")].type}`)
	c.Assert(err, IsNil)
	c.Assert(condType, Equals, "Available")

	condStatus, err := ResolveJsonpathToString(obj, `{.status.conditions[?(@.type == "Available")].status}`)
	c.Assert(err, IsNil)
	c.Assert(condStatus, Equals, "True")

	_, err = ResolveJsonpathToString(obj, "{.status.something}")
	c.Assert(err, NotNil)
}
