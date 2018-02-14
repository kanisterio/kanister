package kube

import (
	"context"
	"fmt"

	batch "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const defautlJobPodName = "kanister-job-pod"
const defaultJobPodContainer = "kanister-job-container"

// Job object is used for running the user specified container as a Kubernetes job.
type Job struct {
	image   string
	command []string

	namespace string
	name      string

	clientset kubernetes.Interface
}

// NewJob creates a new Job object.
func NewJob(clientset kubernetes.Interface, jobName string, namespace string, image string, command ...string) (*Job, error) {
	if jobName == "" {
		return nil, fmt.Errorf("Job name is required")
	}

	if image == "" {
		return nil, fmt.Errorf("Container image needs to be passed")
	}

	if namespace == "" {
		fmt.Printf("No namespace specified. Using \"default\".")
		namespace = "default"
	}

	if clientset == nil {
		return nil, fmt.Errorf("No clientset object provided")
	}

	if len(command) == 0 || command[0] == "" {
		return nil, fmt.Errorf("Command needs to be passed")
	}

	return &Job{image, command, namespace, jobName, clientset}, nil
}

// Create creates the Job in Kubernetes.
func (job *Job) Create() error {
	falseVal := false

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
					Containers: []v1.Container{
						{
							Name:    defaultJobPodContainer,
							Image:   job.image,
							Command: job.command,
							SecurityContext: &v1.SecurityContext{
								Privileged: &falseVal,
							},
							ImagePullPolicy: v1.PullPolicy(v1.PullIfNotPresent),
						},
					},
					RestartPolicy: v1.RestartPolicyOnFailure,
				},
			},
		},
	}

	batchClient := job.clientset.BatchV1()
	jobsClient := batchClient.Jobs(job.namespace)

	newJob, err := jobsClient.Create(k8sJob)
	if err != nil {
		fmt.Printf("Failed to create job: %s\n", err)
		return err
	}
	job.name = newJob.Name
	fmt.Printf("New job %s created\n", job.name)

	return nil
}

// WaitForCompletion waits for the job to run to completion.
func (job *Job) WaitForCompletion(ctx context.Context) error {
	batchClient := job.clientset.BatchV1()
	jobsClient := batchClient.Jobs(job.namespace)
	watch, err := jobsClient.Watch(metav1.ListOptions{LabelSelector: "job-name=" + job.name})
	if err != nil {
		return fmt.Errorf("Failed to create watch object: %s", err)
	}

	// Before getting into the loop of watching events, confirm that the job is actually present
	// in Kubernetes.
	k8sjob, err := jobsClient.Get(job.name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Failed to get job %s", job.name)
	}

	if k8sjob == nil {
		return fmt.Errorf("Couldn't find job %s", job.name)
	}

	events := watch.ResultChan()
	for {
		select {
		case event := <-events:
			if event.Object == nil {
				return fmt.Errorf("Result channel closed for Job %s", job.name)
			}
			k8sJob, ok := event.Object.(*batch.Job)
			if !ok {
				return fmt.Errorf("Invalid Job event object: %T", event.Object)
			}
			conditions := k8sJob.Status.Conditions
			for _, condition := range conditions {
				if condition.Type == batch.JobComplete {
					fmt.Printf("Job %s reported complete\n", job.name)
					return nil
				} else if condition.Type == batch.JobFailed {
					fmt.Printf("Job %s reported Failed\n", job.name)
					return fmt.Errorf("Job %s failed", job.name)
				}
			}
		case <-ctx.Done():
			return fmt.Errorf("Cancellation received")
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
		return fmt.Errorf("Failed to delete job %s: %s", job.name, err)
	}
	fmt.Printf("Deleted job %s\n", job.name)

	return nil
}
