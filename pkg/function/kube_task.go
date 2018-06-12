package function

import (
	"context"
	"math/rand"
	"strconv"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/pkg/errors"
)

const (
	jobPrefix            = "kanister-job-"
	KubeTaskNamespaceArg = "namespace"
	KubeTaskImageArg     = "image"
	KubeTaskCommandArg   = "command"
)

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

func (ktf *kubeTaskFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) error {
	var namespace, image string
	var command []string
	var err error
	if err = Arg(args, KubeTaskNamespaceArg, &namespace); err != nil {
		return err
	}
	if err = Arg(args, KubeTaskImageArg, &image); err != nil {
		return err
	}
	if err = Arg(args, KubeTaskCommandArg, &command); err != nil {
		return err
	}

	namespace, err = kube.GetControllerNamespace()
	if err != nil {
		return errors.Wrapf(err, "Failed to get controller namespace")
	}

	jobName := generateJobName(jobPrefix)
	clientset, err := kube.NewClient()
	if err != nil {
		return errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	serviceAccount, err := kube.GetControllerServiceAccount(clientset)
	if err != nil {
		return errors.Wrap(err, "Failed to get Controller Service Account")
	}
	job, err := kube.NewJob(clientset, jobName, namespace, serviceAccount, image, nil, command...)
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

func (*kubeTaskFunc) RequiredArgs() []string {
	return []string{KubeTaskNamespaceArg, KubeTaskImageArg, KubeTaskCommandArg}
}
