// Copyright 2022 The Kanister Authors.
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

package app

import (
	"context"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
)

type CSISnapshot struct {
	cli        kubernetes.Interface
	namespace  string
	name       string
	deployment appsv1.Deployment
	pvc        v1.PersistentVolumeClaim
}

func NewCSISnapshot(name string) App {
	return &CSISnapshot{
		name: name,
	}
}

func (csi *CSISnapshot) Init(ctx context.Context) error {
	cfg, err := kube.LoadConfig()
	if err != nil {
		return err
	}

	csi.cli, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}

	return nil
}

func (csi *CSISnapshot) Install(ctx context.Context, namespace string) error {
	csi.namespace = namespace
	pvcObj, err := csi.getAppPersistentVolumeClaimObj()
	if err != nil {
		return err
	}
	pvc, err := csi.cli.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, pvcObj, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	log.Print("PVC created successfully", field.M{"app": csi.name, "pvc": pvc.Name})
	csi.pvc = *pvc
	deploymentObj, err := csi.getAppDeploymentObj()
	if err != nil {
		return err
	}
	deployment, err := csi.cli.AppsV1().Deployments(namespace).Create(ctx, deploymentObj, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	csi.deployment = *deployment
	log.Print("Deployment created successfully", field.M{"app": csi.name, "deployment": deployment.Name})
	return nil
}

func (csi *CSISnapshot) Initialize(ctx context.Context) error {
	return nil
}

func (csi *CSISnapshot) Insert(ctx context.Context) error {
	return nil
}

func (csi *CSISnapshot) Reset(ctx context.Context) error {
	log.Print("Resetting the application.", field.M{"app": csi.name})

	removeLogFileCmd := []string{"sh", "-c", "rm /var/log/time.log"}
	stderr, err := csi.execCommand(ctx, removeLogFileCmd)
	if err != nil {
		return errors.Wrapf(err, "Error while deleting log file: %s", stderr)
	}

	log.Print("Reset of the application was successful.", field.M{"app": csi.name})
	return nil
}

func (csi *CSISnapshot) IsReady(ctx context.Context) (bool, error) {
	log.Print("Waiting for the application to be ready.", field.M{"app": csi.name})
	ctx, cancel := context.WithTimeout(ctx, mssqlWaitTimeout)
	defer cancel()

	err := kube.WaitOnDeploymentReady(ctx, csi.cli, csi.namespace, csi.deployment.Name)
	if err != nil {
		return false, err
	}
	log.Print("Application instance is ready.", field.M{"app": csi.name})
	return true, nil
}

func (csi *CSISnapshot) Object() crv1alpha1.ObjectReference {
	return crv1alpha1.ObjectReference{
		Kind:      "deployment",
		Name:      csi.name,
		Namespace: csi.namespace,
	}
}

func (csi *CSISnapshot) Uninstall(ctx context.Context) error {
	err := csi.cli.CoreV1().Namespaces().Delete(ctx, csi.namespace, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	log.Print("Namespace deleted successfully", field.M{"app": csi.name})
	return nil
}

func (csi *CSISnapshot) Ping(ctx context.Context) error {
	log.Print("Pinging the application.", field.M{"app": csi.name})

	listDirectories := []string{"sh", "-c", "ls /var/log"}
	stderr, err := csi.execCommand(ctx, listDirectories)
	if err != nil {
		return errors.Wrapf(err, "Error while Pinging the application %s", stderr)
	}

	log.Print("Ping to the application was success.", field.M{"app": csi.name})
	return nil
}

func (csi *CSISnapshot) GetClusterScopedResources(ctx context.Context) []crv1alpha1.ObjectReference {
	return nil
}

func (csi *CSISnapshot) Count(ctx context.Context) (int, error) {
	return 3, nil
}

func (csi *CSISnapshot) ConfigMaps() map[string]crv1alpha1.ObjectReference {
	return nil
}

func (csi *CSISnapshot) Secrets() map[string]crv1alpha1.ObjectReference {
	return nil
}

func (csi *CSISnapshot) execCommand(ctx context.Context, command []string) (string, error) {
	podname, containername, err := kube.GetPodContainerFromDeployment(ctx, csi.cli, csi.namespace, csi.name)
	if err != nil || podname == "" {
		return "", errors.Wrapf(err, "Error getting pod and containername %s.", csi.name)
	}
	_, stderr, err := kube.Exec(csi.cli, csi.namespace, podname, containername, command, nil)
	return stderr, err
}

func (csi CSISnapshot) getAppDeploymentObj() (*appsv1.Deployment, error) {
	deploymentManifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: time-logger
spec:
  replicas: 1
  selector:
    matchLabels:
      app: time-logger
  template:
    metadata:
      labels:
        app: time-logger
    spec:
      containers:
      - name: test-container
        image: ghcr.io/kanisterio/kanister-tools:0.72.0
        command: ["sh", "-c"]
        args: ["while true; do for x in $(seq 1200); do date >> /var/log/time.log; sleep 1; done; truncate /var/log/time.log --size 0; done"]
        volumeMounts:
        - name: data
          mountPath: /var/log
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: time-log-pvc`

	var deployment *appsv1.Deployment
	err := yaml.Unmarshal([]byte(deploymentManifest), &deployment)
	return deployment, err
}

func (csi CSISnapshot) getAppPersistentVolumeClaimObj() (*v1.PersistentVolumeClaim, error) {
	pvcManifest := `apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: time-log-pvc
  labels:
    app: time-logger
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi`

	var pvc *v1.PersistentVolumeClaim
	err := yaml.Unmarshal([]byte(pvcManifest), &pvc)
	return pvc, err
}
