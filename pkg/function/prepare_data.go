package function

import (
	"context"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	PrepareDataNamespaceArg   = "namespace"
	PrepareDataImageArg       = "image"
	PrepareDataCommandArg     = "command"
	PrepareDataVolumes        = "volumes"
	PrepareDataServiceAccount = "serviceaccount"
)

func init() {
	kanister.Register(&prepareDataFunc{})
}

var _ kanister.Func = (*prepareDataFunc)(nil)

type prepareDataFunc struct{}

func (*prepareDataFunc) Name() string {
	return "PrepareData"
}

func (*prepareDataFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) error {
	var namespace, image, serviceAccount string
	var command []string
	var vols map[string]string
	var err error
	if err = Arg(args, PrepareDataNamespaceArg, &namespace); err != nil {
		return err
	}
	if err = Arg(args, PrepareDataImageArg, &image); err != nil {
		return err
	}
	if err = Arg(args, PrepareDataCommandArg, &command); err != nil {
		return err
	}
	if err = Arg(args, PrepareDataVolumes, &vols); err != nil {
		return err
	}
	if err = OptArg(args, PrepareDataServiceAccount, &serviceAccount, ""); err != nil {
		return err
	}
	cli := kube.NewClient()
	// Validate volumes
	for pvc := range vols {
		if _, err := cli.CoreV1().PersistentVolumeClaims(namespace).Get(pvc, metav1.GetOptions{}); err != nil {
			return errors.Wrapf(err, "Failed to retrieve PVC. Namespace %s, Name %s", namespace, pvc)
		}
	}
	jobName := generateJobName(jobPrefix)
	job, err := kube.NewJob(cli, jobName, namespace, serviceAccount, image, vols, command...)
	if err != nil {
		return errors.Wrap(err, "Failed to create prepare data job")
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

func (*prepareDataFunc) RequiredArgs() []string {
	return []string{PrepareDataNamespaceArg, PrepareDataImageArg, PrepareDataCommandArg, PrepareDataVolumes}
}
