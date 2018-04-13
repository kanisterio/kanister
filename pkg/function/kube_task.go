package function

import (
	"context"
	"math/rand"
	"strconv"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/pkg/errors"
)

const jobPrefix = "kanister-job-"

func init() {
	kanister.Register(&kubeTaskFunc{})
}

var _ kanister.Func = (*kubeTaskFunc)(nil)

type kubeTaskFunc struct{}

func (*kubeTaskFunc) Name() string {
	return "KubeTask"
}

func generateJobName(jobPrefix string) string {
	jobNameSuffix := strconv.FormatUint(rand.Uint64(), 10)
	return jobPrefix + jobNameSuffix
}

func (*kubeTaskFunc) Exec(ctx context.Context, args ...string) error {
	if len(args) <= 2 {
		return errors.Errorf("kubeTaskFunc requires at least 2 arguments. Got: %#v", args)
	}

	image, command := args[0], args[1:]
	namespace, err := kube.GetControllerNamespace()
	if err != nil {
		return errors.Wrapf(err, "Failed to get controller namespace")
	}

	jobName := generateJobName(jobPrefix)
	clientset := kube.NewClient()
	serviceAccount, err := kube.GetControllerServiceAccount(clientset)
	if err != nil {
		return errors.Wrap(err, "Failed to get Controller Service Account")
	}
	job, err := kube.NewJob(clientset, jobName, namespace, serviceAccount, image, command...)
	if err != nil {
		return errors.Wrap(err, "Failed to create job")
	}
	if err := job.Create(); err != nil {
		return errors.Wrapf(err, "Failed to create job %s in Kubernetes", jobName)
	}
	defer job.Delete()
	if err := job.WaitForCompletion(ctx); err != nil {
		return errors.Wrapf(err, "Failed while waiting for job %s to complete", jobName)
	}
	return nil
}
