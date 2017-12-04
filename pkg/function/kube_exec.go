package function

import (
	"context"

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
		log.Info(stdout)
	}
	if stderr != "" {
		log.Info(stderr)
	}
	return err
}
