package function

import (
	"context"
	"regexp"
	"strings"

	"github.com/pkg/errors"
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

type kubeExecFunc struct{}

func (*kubeExecFunc) Name() string {
	return "KubeExec"
}

func (*kubeExecFunc) Exec(ctx context.Context, args ...string) error {
	if len(args) <= 4 {
		return errors.Errorf("KubeExec requires at least 4 arguments. Got: %#v", args)
	}
	cli := kube.NewClient()
	namespace, pod, container, cmd := args[0], args[1], args[2], args[3:]
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
