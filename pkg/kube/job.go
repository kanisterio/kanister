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
	"github.com/kanisterio/errkit"
	batch "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
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
	vols      map[string]VolumeMountOptions
	clientset kubernetes.Interface
}

// NewJob creates a new Job object.
func NewJob(clientset kubernetes.Interface,
	jobName string,
	namespace string,
	serviceAccount string,
	image string,
	vols map[string]VolumeMountOptions,
	command ...string,
) (*Job, error) {
	if jobName == "" {
		return nil, errkit.New("Job name is required")
	}

	if image == "" {
		return nil, errkit.New("Container image needs to be passed")
	}

	if namespace == "" {
		log.Debug().Print("No namespace specified. Using \"default\".")
		namespace = "default"
	}

	if clientset == nil {
		return nil, errkit.New("No clientset object provided")
	}

	if len(command) == 0 || command[0] == "" {
		return nil, errkit.New("Command needs to be passed")
	}

	return &Job{image, command, namespace, jobName, serviceAccount, vols, clientset}, nil
}

// Create creates the Job in Kubernetes.
func (job *Job) Create() error {
	falseVal := false
	ctx := context.TODO()
	volumeMounts, podVolumes, err := createFilesystemModeVolumeSpecs(ctx, job.vols)
	if err != nil {
		return errkit.Wrap(err, "Failed to create volume spec for job", "jobName", job.name)
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
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   defautlJobPodName,
					Labels: make(map[string]string),
				},
				Spec: v1.PodSpec{
					ServiceAccountName: job.sa,
					Containers: []v1.Container{
						{
							Name:    defaultJobPodContainer,
							Image:   job.image,
							Command: job.command,
							SecurityContext: &v1.SecurityContext{
								Privileged: &falseVal,
							},
							ImagePullPolicy: v1.PullPolicy(v1.PullIfNotPresent),
							VolumeMounts:    volumeMounts,
						},
					},
					RestartPolicy: v1.RestartPolicyOnFailure,
					Volumes:       podVolumes,
				},
			},
		},
	}

	batchClient := job.clientset.BatchV1()
	jobsClient := batchClient.Jobs(job.namespace)

	newJob, err := jobsClient.Create(ctx, k8sJob, metav1.CreateOptions{})
	if err != nil {
		return errkit.Wrap(err, "Failed to create job", "jobName", job.name)
	}
	job.name = newJob.Name
	log.Print("New job created", field.M{"JobName": job.name})

	return nil
}

func createFilesystemModeVolumeSpecs(
	ctx context.Context,
	vols map[string]VolumeMountOptions,
) (volumeMounts []v1.VolumeMount, podVolumes []v1.Volume, error error) {
	// Build filesystem mode volume specs
	for pvcName, mountOpts := range vols {
		id, err := uuid.NewV1()
		if err != nil {
			return nil, nil, err
		}

		if mountOpts.ReadOnly {
			log.Debug().WithContext(ctx).Print("PVC will be mounted in read-only mode", field.M{"pvcName": pvcName})
		}

		podVolName := fmt.Sprintf("vol-%s", id.String())
		volumeMounts = append(volumeMounts, v1.VolumeMount{Name: podVolName, MountPath: mountOpts.MountPath, ReadOnly: mountOpts.ReadOnly})
		podVolumes = append(podVolumes,
			v1.Volume{
				Name: podVolName,
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						ClaimName: pvcName,
						ReadOnly:  mountOpts.ReadOnly,
					},
				},
			},
		)
	}
	return volumeMounts, podVolumes, nil
}

func createBlockModeVolumeSpecs(blockVols map[string]string) (volumeDevices []v1.VolumeDevice, podVolumes []v1.Volume, error error) {
	// Build block mode volume specs
	for pvc, devicePath := range blockVols {
		id, err := uuid.NewV1()
		if err != nil {
			return nil, nil, err
		}
		podBlockVolName := fmt.Sprintf("block-%s", id.String())
		volumeDevices = append(volumeDevices, v1.VolumeDevice{Name: podBlockVolName, DevicePath: devicePath})
		podVolumes = append(podVolumes,
			v1.Volume{
				Name: podBlockVolName,
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
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
		return errkit.Wrap(err, "Failed to create watch object")
	}

	// Before getting into the loop of watching events, confirm that the job is actually present
	// in Kubernetes.
	k8sjob, err := jobsClient.Get(ctx, job.name, metav1.GetOptions{})
	if err != nil {
		return errkit.Wrap(err, "Failed to get job", "jobName", job.name)
	}

	if k8sjob == nil {
		return errkit.Wrap(err, "Couldn't find job", "jobName", job.name)
	}

	events := watch.ResultChan()
	for {
		select {
		case event := <-events:
			if event.Object == nil {
				return errkit.New("Result channel closed for Job", "jobName", job.name)
			}
			k8sJob, ok := event.Object.(*batch.Job)
			if !ok {
				return errkit.New("Invalid Job event object", "eventType", fmt.Sprintf("%T", event.Object))
			}
			conditions := k8sJob.Status.Conditions
			for _, condition := range conditions {
				if condition.Type == batch.JobComplete {
					log.Print("Job reported complete\n", field.M{"JobName": job.name})
					return nil
				} else if condition.Type == batch.JobFailed {
					return errkit.New("Job failed", "jobName", job.name)
				}
			}
		case <-ctx.Done():
			return errkit.New("Cancellation received")
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
		return errkit.Wrap(err, "Failed to delete job", "jobName", job.name)
	}
	log.Print("Deleted job", field.M{"JobName": job.name})

	return nil
}
