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
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	batch "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

const defautlJobPodName = "kanister-job-pod"
const defaultJobPodContainer = "kanister-job-container"

// Job object is used for running the user specified container as a Kubernetes job.
type Job struct {
	image     string
	command   []string
	namespace string
	name      string
	sa        string
	// vols is a map of PVC->Mount points to add to the job pod spec
	vols      map[string]string
	clientset kubernetes.Interface
}

// NewJob creates a new Job object.
func NewJob(clientset kubernetes.Interface, jobName string, namespace string, serviceAccount string, image string, vols map[string]string, command ...string) (*Job, error) {
	if jobName == "" {
		return nil, errors.New("Job name is required")
	}

	if image == "" {
		return nil, errors.New("Container image needs to be passed")
	}

	if namespace == "" {
		log.Debug().Print("No namespace specified. Using \"default\".")
		namespace = "default"
	}

	if clientset == nil {
		return nil, errors.New("No clientset object provided")
	}

	if len(command) == 0 || command[0] == "" {
		return nil, errors.New("Command needs to be passed")
	}

	return &Job{image, command, namespace, jobName, serviceAccount, vols, clientset}, nil
}

// Create creates the Job in Kubernetes.
func (job *Job) Create() error {
	falseVal := false
	volumeMounts, podVolumes, err := createFilesystemModeVolumeSpecs(job.vols)
	if err != nil {
		return errors.Wrapf(err, "Failed to create volume spec for job %s", job.name)
	}
	k8sJob := &batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: job.name,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "v1",
		},
		Spec: batch.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   defautlJobPodName,
					Labels: make(map[string]string),
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: job.sa,
					Containers: []corev1.Container{
						{
							Name:    defaultJobPodContainer,
							Image:   job.image,
							Command: job.command,
							SecurityContext: &corev1.SecurityContext{
								Privileged: &falseVal,
							},
							ImagePullPolicy: corev1.PullPolicy(corev1.PullIfNotPresent),
							VolumeMounts:    volumeMounts,
						},
					},
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Volumes:       podVolumes,
				},
			},
		},
	}

	batchClient := job.clientset.BatchV1()
	jobsClient := batchClient.Jobs(job.namespace)

	newJob, err := jobsClient.Create(context.TODO(), k8sJob, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrapf(err, "Failed to create job %s", job.name)
	}
	job.name = newJob.Name
	log.Print("New job created", field.M{"JobName": job.name})

	return nil
}

func createFilesystemModeVolumeSpecs(vols map[string]string) (volumeMounts []corev1.VolumeMount, podVolumes []corev1.Volume, error error) {
	// Build filesystem mode volume specs
	for pvc, mountPath := range vols {
		id, err := uuid.NewV1()
		if err != nil {
			return nil, nil, err
		}
		podVolName := fmt.Sprintf("vol-%s", id.String())
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: podVolName, MountPath: mountPath})
		podVolumes = append(podVolumes,
			corev1.Volume{
				Name: podVolName,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: pvc,
					},
				},
			},
		)
	}
	return volumeMounts, podVolumes, nil
}

func createBlockModeVolumeSpecs(blockVols map[string]string) (volumeDevices []corev1.VolumeDevice, podVolumes []corev1.Volume, error error) {
	// Build block mode volume specs
	for pvc, devicePath := range blockVols {
		id, err := uuid.NewV1()
		if err != nil {
			return nil, nil, err
		}
		podBlockVolName := fmt.Sprintf("block-%s", id.String())
		volumeDevices = append(volumeDevices, corev1.VolumeDevice{Name: podBlockVolName, DevicePath: devicePath})
		podVolumes = append(podVolumes,
			corev1.Volume{
				Name: podBlockVolName,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: pvc,
					},
				},
			},
		)
	}
	return volumeDevices, podVolumes, nil
}

// WaitForCompletion waits for the job to run to completion.
func (job *Job) WaitForCompletion(ctx context.Context) error {
	batchClient := job.clientset.BatchV1()
	jobsClient := batchClient.Jobs(job.namespace)
	watch, err := jobsClient.Watch(ctx, metav1.ListOptions{LabelSelector: "job-name=" + job.name})
	if err != nil {
		return errors.Wrap(err, "Failed to create watch object")
	}

	// Before getting into the loop of watching events, confirm that the job is actually present
	// in Kubernetes.
	k8sjob, err := jobsClient.Get(ctx, job.name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "Failed to get job %s", job.name)
	}

	if k8sjob == nil {
		return errors.Wrapf(err, "Couldn't find job %s", job.name)
	}

	events := watch.ResultChan()
	for {
		select {
		case event := <-events:
			if event.Object == nil {
				return errors.Errorf("Result channel closed for Job %s", job.name)
			}
			k8sJob, ok := event.Object.(*batch.Job)
			if !ok {
				return errors.Errorf("Invalid Job event object: %T", event.Object)
			}
			conditions := k8sJob.Status.Conditions
			for _, condition := range conditions {
				if condition.Type == batch.JobComplete {
					log.Print("Job reported complete\n", field.M{"JobName": job.name})
					return nil
				} else if condition.Type == batch.JobFailed {
					return errors.Errorf("Job %s failed", job.name)
				}
			}
		case <-ctx.Done():
			return errors.New("Cancellation received")
		}
	}
}

// Delete deletes the Job in Kubernetes.
func (job *Job) Delete() error {
	batchClient := job.clientset.BatchV1()
	jobsClient := batchClient.Jobs(job.namespace)
	deletePropagation := metav1.DeletePropagationForeground
	err := jobsClient.Delete(context.TODO(), job.name, metav1.DeleteOptions{PropagationPolicy: &deletePropagation})
	if err != nil {
		return errors.Wrapf(err, "Failed to delete job %s", job.name)
	}
	log.Print("Deleted job", field.M{"JobName": job.name})

	return nil
}
