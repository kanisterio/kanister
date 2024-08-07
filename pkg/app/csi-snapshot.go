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

	"github.com/kanisterio/errkit"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
)

// TimeLogCSI is integration test app for CSI Snapshot functions.
type TimeLogCSI struct {
	cli        kubernetes.Interface
	namespace  string
	name       string
	deployment appsv1.Deployment
	pvc        corev1.PersistentVolumeClaim
}

// NewTimeLogCSI instantiates the TimeLogCSI integration test app
func NewTimeLogCSI(name string) App {
	return &TimeLogCSI{
		name: name,
	}
}

// Init initialises kubernetes CLI
func (tlc *TimeLogCSI) Init(ctx context.Context) error {
	cfg, err := kube.LoadConfig()
	if err != nil {
		return err
	}
	tlc.cli, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}
	return nil
}

// Install deploys the TimeLogCSI App Deployment and PVC
func (tlc *TimeLogCSI) Install(ctx context.Context, namespace string) error {
	tlc.namespace = namespace
	pvcObj := tlc.getAppPersistentVolumeClaimObj()
	pvc, err := tlc.cli.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, pvcObj, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	log.Print("PVC created successfully", field.M{"app": tlc.name, "pvc": pvc.Name})
	tlc.pvc = *pvc
	deploymentObj := tlc.getAppDeploymentObj()
	deployment, err := tlc.cli.AppsV1().Deployments(namespace).Create(ctx, deploymentObj, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	tlc.deployment = *deployment
	log.Print("Deployment created successfully", field.M{"app": tlc.name, "deployment": deployment.Name})
	return nil
}

// Reset deletes the log file present at the volume mount path
func (tlc *TimeLogCSI) Reset(ctx context.Context) error {
	log.Print("Resetting the application.", field.M{"app": tlc.name})

	removeLogFileCmd := []string{"sh", "-c", "rm /var/log/time.log"}
	stderr, err := tlc.execCommand(ctx, removeLogFileCmd)
	if err != nil {
		return errkit.Wrap(err, "Error while deleting log file", "stderr", stderr)
	}

	log.Print("Reset of the application was successful.", field.M{"app": tlc.name})
	return nil
}

// IsReady waits for the App Deployment to be in 'Ready' state
func (tlc *TimeLogCSI) IsReady(ctx context.Context) (bool, error) {
	log.Print("Waiting for the application to be ready.", field.M{"app": tlc.name})
	ctx, cancel := context.WithTimeout(ctx, mssqlWaitTimeout)
	defer cancel()

	err := kube.WaitOnDeploymentReady(ctx, tlc.cli, tlc.namespace, tlc.deployment.Name)
	if err != nil {
		return false, err
	}
	log.Print("Application instance is ready.", field.M{"app": tlc.name})
	return true, nil
}

// Object defines the objectReference that will be used to create actions in blueprint
func (tlc *TimeLogCSI) Object() crv1alpha1.ObjectReference {
	return crv1alpha1.ObjectReference{
		Kind:      "deployment",
		Name:      tlc.name,
		Namespace: tlc.namespace,
	}
}

// Uninstall removes the TimeLogCSI app from the cluster
func (tlc *TimeLogCSI) Uninstall(ctx context.Context) error {
	err := tlc.cli.CoreV1().Namespaces().Delete(ctx, tlc.namespace, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	log.Print("Namespace deleted successfully", field.M{"app": tlc.name})
	return nil
}

// Ping is used to check the connection with Deployment Pod
func (tlc *TimeLogCSI) Ping(ctx context.Context) error {
	log.Print("Pinging the application.", field.M{"app": tlc.name})

	listDirectories := []string{"sh", "-c", "ls /var/log"}
	stderr, err := tlc.execCommand(ctx, listDirectories)
	if err != nil {
		return errkit.Wrap(err, "Error while Pinging the application", "stderr", stderr)
	}

	log.Print("Ping to the application was success.", field.M{"app": tlc.name})
	return nil
}

func (tlc *TimeLogCSI) Insert(ctx context.Context) error {
	return nil
}

func (tlc *TimeLogCSI) Initialize(ctx context.Context) error {
	return nil
}

func (tlc *TimeLogCSI) Count(ctx context.Context) (int, error) {
	return 3, nil
}

func (tlc *TimeLogCSI) Secrets() map[string]crv1alpha1.ObjectReference {
	return nil
}

func (tlc *TimeLogCSI) ConfigMaps() map[string]crv1alpha1.ObjectReference {
	return nil
}

func (tlc *TimeLogCSI) GetClusterScopedResources(ctx context.Context) []crv1alpha1.ObjectReference {
	return nil
}

func (tlc *TimeLogCSI) execCommand(ctx context.Context, command []string) (string, error) {
	podname, containername, err := kube.GetPodContainerFromDeployment(ctx, tlc.cli, tlc.namespace, tlc.name)
	if err != nil || podname == "" {
		return "", errkit.Wrap(err, "Error getting pod and containername.", "deployment", tlc.name)
	}
	_, stderr, err := kube.Exec(ctx, tlc.cli, tlc.namespace, podname, containername, command, nil)
	return stderr, err
}

func (tlc TimeLogCSI) getAppDeploymentObj() *appsv1.Deployment {
	depReplicas := int32(1)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: tlc.name,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &depReplicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": tlc.name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": tlc.name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "test-container",
							Image:   "ghcr.io/kanisterio/kanister-tools:0.110.0",
							Command: []string{"sh", "-c"},
							Args:    []string{"while true; do for x in $(seq 1200); do date >> /var/log/time.log; sleep 1; done; truncate /var/log/time.log --size 0; done"},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data",
									MountPath: "/var/log",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "time-log-pvc",
								},
							},
						},
					},
				},
			},
		},
	}
	return deployment
}

func (tlc TimeLogCSI) getAppPersistentVolumeClaimObj() *corev1.PersistentVolumeClaim {
	storageClassName := "csi-hostpath-sc"
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "time-log-pvc",
			Labels: map[string]string{"app": tlc.name},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: &storageClassName,
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: *resource.NewQuantity(1073741824, resource.BinarySI),
				},
			},
		},
	}
	return pvc
}
