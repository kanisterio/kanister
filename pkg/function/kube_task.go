package function

import (
	"context"
	"math/rand"
	"strconv"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/format"
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

func kubeTask(ctx context.Context, namespace, image string, command []string) (map[string]interface{}, error) {
	var serviceAccount string
	var err error
	clientset, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	if namespace == "" {
		namespace, err = kube.GetControllerNamespace()
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to get controller namespace")
		}
		serviceAccount, err = kube.GetControllerServiceAccount(clientset)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get Controller Service Account")
		}
	}
	// Create a pod to run the command
	pod, err := kube.CreatePod(ctx, clientset, &kube.PodOptions{
		Namespace:          namespace,
		GenerateName:       jobPrefix,
		Image:              image,
		Command:            command,
		ServiceAccountName: serviceAccount,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create pod for KubeTask")
	}
	defer kube.DeletePod(context.Background(), clientset, pod)

	// Wait for pod completion
	if err := kube.WaitForPodCompletion(ctx, clientset, pod.Namespace, pod.Name); err != nil {
		return nil, errors.Wrapf(err, "Failed while waiting for Pod %s to complete", pod.Name)
	}
	// Fetch logs from the pod
	logs, err := kube.GetPodLogs(ctx, clientset, pod.Namespace, pod.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to fetch logs from the pod")
	}
	format.Log(pod.Name, pod.Spec.Containers[0].Name, logs)

	out, err := parseLogAndCreateOutput(logs)
	return out, errors.Wrap(err, "Failed to generate output")
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
	return kubeTask(ctx, namespace, image, command)
}

func (*kubeTaskFunc) RequiredArgs() []string {
	return []string{KubeTaskNamespaceArg, KubeTaskImageArg, KubeTaskCommandArg}
}
