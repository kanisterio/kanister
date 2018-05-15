// +build !unit

package kube

import (
	"context"
	"fmt"
	"time"

	"github.com/jpillora/backoff"
	. "gopkg.in/check.v1"
	batch "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

type JobSuite struct{}

var _ = Suite(&JobSuite{})

const testJobName = "kanister-test-job"
const testJobNamespace = "default"
const testJobImage = "busybox"
const testJobServiceAccount = "default"

func (s *JobSuite) SetUpSuite(c *C) {
	// c.Skip("Too slow")
}

// Verifies that the Job object is not created if the job name is not specified.
func (s *JobSuite) TestJobsNoName(c *C) {
	clientset := NewClient()

	job, err := NewJob(clientset, testJobName, testJobNamespace, testJobServiceAccount, "", nil, "sleep", "10")
	c.Assert(job, IsNil)
	c.Assert(err, NotNil)
}

// Verifies that the Job object is not created if the image is not specified.
func (s *JobSuite) TestJobsNoImage(c *C) {
	clientset := NewClient()

	job, err := NewJob(clientset, testJobName, testJobNamespace, testJobServiceAccount, "", nil, "sleep", "10")
	c.Assert(job, IsNil)
	c.Assert(err, NotNil)
}

// Verifies that the Job object is not created if the namespace is not specified.
func (s *JobSuite) TestJobsNoNamespace(c *C) {
	clientset := NewClient()

	job, err := NewJob(clientset, testJobName, "", testJobServiceAccount, testJobImage, nil, "sleep", "10")
	c.Assert(job.namespace, Equals, "default")
	c.Assert(err, IsNil)
}

// Verifies that the Job object is not created if the clientset is nil.
func (s *JobSuite) TestJobsNoClientset(c *C) {
	job, err := NewJob(nil, testJobName, testJobNamespace, testJobServiceAccount, testJobImage, nil, "sleep", "10")
	c.Assert(job, IsNil)
	c.Assert(err, NotNil)
}

// Verifies that the Job object is not created if the command is not passed.
func (s *JobSuite) TestJobsNoCommand(c *C) {
	clientset := NewClient()

	job, err := NewJob(clientset, testJobName, testJobNamespace, testJobServiceAccount, testJobImage, nil, "")
	c.Assert(job, IsNil)
	c.Assert(err, NotNil)

	job, err = NewJob(clientset, testJobName, testJobNamespace, testJobServiceAccount, testJobImage, nil)
	c.Assert(job, IsNil)
	c.Assert(err, NotNil)
}

func getK8sJobCount(clientset kubernetes.Interface, namespace string, c *C) int {
	jobsCli := clientset.BatchV1().Jobs(namespace)
	list, err := jobsCli.List(metav1.ListOptions{LabelSelector: "job-name=" + testJobName})
	c.Assert(err, IsNil)

	return len(list.Items)
}

func waitForJobCount(clientset kubernetes.Interface, namespace string, expectedCount int, c *C) error {
	// At times, even if the job is deleted, the API server takes sometime
	// to consolidate it's state.
	maxRetries := 10
	boff := &backoff.Backoff{Factor: 2, Jitter: false, Min: 100 * time.Millisecond, Max: 1 * time.Minute}
	var newJobCount int
	for {
		newJobCount = getK8sJobCount(clientset, namespace, c)
		if newJobCount != expectedCount {
			if int(boff.Attempt()) >= maxRetries {
				return fmt.Errorf("Job count %d, expected job count %d", newJobCount, expectedCount)
			}
			duration := boff.Duration()
			fmt.Printf("Trying again in %s\n", duration)
			time.Sleep(duration)
			continue
		} else {
			return nil
		}
	}
}

// Verifies that the basic Job creation and deletion completes successfully.
func (s *JobSuite) TestJobsBasic(c *C) {
	namespace := "default"
	clientset := NewClient()

	origJobCount := getK8sJobCount(clientset, namespace, c)

	images := [2]string{"ubuntu:latest", "perl"}
	for _, image := range images {
		job, err := NewJob(clientset, testJobName, testJobNamespace, testJobServiceAccount, image, nil, "sleep", "2")

		c.Assert(job, NotNil)
		c.Assert(err, IsNil)

		err = job.Create()
		c.Assert(err, IsNil)

		ctx := context.Background()
		err = job.WaitForCompletion(ctx)
		c.Assert(err, IsNil)

		err = job.Delete()
		c.Assert(err, IsNil)

		err = waitForJobCount(clientset, namespace, origJobCount, c)
		if err != nil {
			c.Fail()
		}
	}
}

// Verifies that deleting the Job while it is running works.
func (s *JobSuite) TestJobsDeleteWhileRunning(c *C) {
	namespace := "default"
	clientset := NewClient()

	job, err := NewJob(clientset, testJobName, testJobNamespace, testJobServiceAccount, testJobImage, nil, "sleep", "300")

	c.Assert(job, NotNil)
	c.Assert(err, IsNil)

	origJobCount := getK8sJobCount(clientset, namespace, c)
	// Start the job that will run for 5 minutes
	job.Create()
	time.Sleep(100 * time.Millisecond)
	// Deleting the job should work.
	job.Delete()

	err = waitForJobCount(clientset, namespace, origJobCount, c)
	c.Assert(c, NotNil)
}

func cancelLater(cancel func()) {
	time.Sleep(10 * time.Second)
	cancel()
}

// Verifies that cancelling the context results in WaitForCompletion finishing.
func (s *JobSuite) TestJobsWaitAfterDelete(c *C) {
	clientset := NewClient()

	job, err := NewJob(clientset, testJobName, testJobNamespace, testJobServiceAccount, testJobImage, nil, "sleep", "300")

	c.Assert(job, NotNil)
	c.Assert(err, IsNil)

	// Start the job and then delete it!
	job.Create()
	job.Delete()
	// Wait for the job to be completely deleted.
	time.Sleep(10 * time.Second)
	jobCount := getK8sJobCount(clientset, testJobNamespace, c)
	c.Assert(jobCount, Equals, 0)

	ctx, cancel := context.WithCancel(context.Background())
	go cancelLater(cancel)
	// WaitForCompletion should complete when the context is cancelled.
	err = job.WaitForCompletion(ctx)
	c.Assert(c, NotNil)
}

func (s *JobSuite) TestJobsWaitOnNonExistentJob(c *C) {
	clientset := NewClient()

	job, err := NewJob(clientset, testJobName, testJobNamespace, testJobServiceAccount, testJobImage, nil, "sleep", "300")

	c.Assert(job, NotNil)
	c.Assert(err, IsNil)

	// Call WaitForCompletion on non-existent kubernetes job.
	err = job.WaitForCompletion(context.Background())
	c.Assert(c, NotNil)
}

func (s *JobSuite) TestJobsVolumes(c *C) {
	cli := fake.NewSimpleClientset()
	vols := map[string]string{"pvc-test": "/mnt/data1"}
	job, err := NewJob(cli, testJobName, testJobNamespace, testJobServiceAccount, testJobImage, vols, "sleep", "300")
	c.Assert(err, IsNil)
	c.Assert(job.Create(), IsNil)

	a := cli.Actions()
	c.Assert(a, HasLen, 1)
	createAction := a[0]
	createdJob, ok := createAction.(k8stesting.CreateAction).GetObject().(*batch.Job)
	c.Assert(ok, Equals, true)

	c.Assert(createdJob.Name, Equals, testJobName)
	podSpec := createdJob.Spec.Template.Spec
	c.Assert(podSpec.Volumes, HasLen, 1)
	c.Assert(podSpec.Volumes[0].VolumeSource.PersistentVolumeClaim.ClaimName, Equals, "pvc-test")
	c.Assert(podSpec.Containers[0].VolumeMounts[0].MountPath, Equals, "/mnt/data1")
}
