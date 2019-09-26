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

	"github.com/pkg/errors"
	batch "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

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
	volumeMounts, podVolumes := createVolumeSpecs(job.vols)
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

	newJob, err := jobsClient.Create(k8sJob)
	if err != nil {
		return errors.Wrapf(err, "Failed to create job %s", job.name)
	}
	job.name = newJob.Name
	log.Print(fmt.Sprintf("New job %s created", job.name))

	return nil
}

func createVolumeSpecs(vols map[string]string) (volumeMounts []v1.VolumeMount, podVolumes []v1.Volume) {
	// Build volume specs
	for pvc, mountPath := range vols {
		podVolName := fmt.Sprintf("vol-%s", pvc)
		volumeMounts = append(volumeMounts, v1.VolumeMount{Name: podVolName, MountPath: mountPath})
		podVolumes = append(podVolumes,
			v1.Volume{
				Name: podVolName,
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						ClaimName: pvc,
					},
				},
			},
		)
	}
	return volumeMounts, podVolumes
}

// WaitForCompletion waits for the job to run to completion.
func (job *Job) WaitForCompletion(ctx context.Context) error {
	batchClient := job.clientset.BatchV1()
	jobsClient := batchClient.Jobs(job.namespace)
	watch, err := jobsClient.Watch(metav1.ListOptions{LabelSelector: "job-name=" + job.name})
	if err != nil {
		return errors.Wrap(err, "Failed to create watch object")
	}

	// Before getting into the loop of watching events, confirm that the job is actually present
	// in Kubernetes.
	k8sjob, err := jobsClient.Get(job.name, metav1.GetOptions{})
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
					log.Print(fmt.Sprintf("Job %s reported complete\n", job.name))
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
	var deletePropagation metav1.DeletionPropagation
	deletePropagation = metav1.DeletePropagationForeground
	err := jobsClient.Delete(job.name, &metav1.DeleteOptions{PropagationPolicy: &deletePropagation})
	if err != nil {
		return errors.Wrapf(err, "Failed to delete job %s", job.name)
	}
	log.Print(fmt.Sprintf("Deleted job %s", job.name))

	return nil
}
