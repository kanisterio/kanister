package function

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
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

func kubeTask(ctx context.Context, cli kubernetes.Interface, namespace, image string, command []string) (map[string]interface{}, error) {
	var serviceAccount string
	var err error
	if namespace == "" {
		namespace, err = kube.GetControllerNamespace()
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to get controller namespace")
		}
		serviceAccount, err = kube.GetControllerServiceAccount(cli)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get Controller Service Account")
		}
	}
	options := &kube.PodOptions{
		Namespace:          namespace,
		GenerateName:       jobPrefix,
		Image:              image,
		Command:            command,
		ServiceAccountName: serviceAccount,
	}
	pr := kube.NewPodRunner(cli, options)
	podFunc := kubeTaskPodFunc(cli)
	return pr.Run(ctx, podFunc)
}

func kubeTaskPodFunc(cli kubernetes.Interface) func(ctx context.Context, pod *v1.Pod) (map[string]interface{}, error) {
	return func(ctx context.Context, pod *v1.Pod) (map[string]interface{}, error) {
		// Wait for pod completion
		if err := kube.WaitForPodCompletion(ctx, cli, pod.Namespace, pod.Name); err != nil {
			return nil, errors.Wrapf(err, "Failed while waiting for Pod %s to complete", pod.Name)
		}
		// Fetch logs from the pod
		logs, err := kube.GetPodLogs(ctx, cli, pod.Namespace, pod.Name)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to fetch logs from the pod")
		}
		format.Log(pod.Name, pod.Spec.Containers[0].Name, logs)
		out, err := parseLogAndCreateOutput(logs)
		return out, errors.Wrap(err, "Failed to parse phase output")
	}
}

func (ktf *kubeTaskFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var namespace, image string
	var command []string
	var err error
	if err = Arg(args, KubeTaskImageArg, &image); err != nil {
		return nil, err
	}
	if err = Arg(args, KubeTaskCommandArg, &command); err != nil {
		return nil, err
	}
	if err = OptArg(args, KubeTaskNamespaceArg, &namespace, ""); err != nil {
		return nil, err
	}
	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	return kubeTask(ctx, cli, namespace, image, command)
}

func (*kubeTaskFunc) RequiredArgs() []string {
	return []string{KubeTaskNamespaceArg, KubeTaskImageArg, KubeTaskCommandArg}
}
