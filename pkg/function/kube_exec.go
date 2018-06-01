package function

import (
	"context"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/kube"
)

func init() {
	kanister.Register(&kubeExecFunc{})
}

var (
	_ kanister.Func = (*kubeExecFunc)(nil)
)

const (
	KubeExecNamespaceArg     = "namespace"
	KubeExecPodNameArg       = "pod"
	KubeExecContainerNameArg = "container"
	KubeExecCommandArg       = "command"
)

type kubeExecFunc struct{}

func (*kubeExecFunc) Name() string {
	return "KubeExec"
}

func (kef *kubeExecFunc) Exec(ctx context.Context, args map[string]interface{}) error {
	cli := kube.NewClient()
	var namespace, pod, container string
	var cmd []string
	var err error
	if err = Arg(args, KubeExecNamespaceArg, &namespace); err != nil {
		return err
	}
	if err = Arg(args, KubeExecPodNameArg, &pod); err != nil {
		return err
	}
	if err = Arg(args, KubeExecContainerNameArg, &container); err != nil {
		return err
	}
	if err = Arg(args, KubeExecCommandArg, &cmd); err != nil {
		return err
	}

	stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd)
	if stdout != "" {
		logs := regexp.MustCompile("[\r\n]").Split(stdout, -1)
		for _, stdoutLog := range logs {
			if strings.TrimSpace(stdoutLog) != "" {
				log.Info(stdoutLog)
			}
		}
	}
	if stderr != "" {
		logs := regexp.MustCompile("[\r\n]").Split(stderr, -1)
		for _, stderrLog := range logs {
			if strings.TrimSpace(stderrLog) != "" {
				log.Info(stderrLog)
			}
		}
	}
	return err
}

func (*kubeExecFunc) RequiredArgs() []string {
	return []string{KubeExecNamespaceArg, KubeExecPodNameArg, KubeExecContainerNameArg, KubeExecCommandArg}
}
