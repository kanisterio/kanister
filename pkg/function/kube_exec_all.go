package function

import (
	"context"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
)

func init() {
	kanister.Register(&kubeExecAllFunc{})
}

var (
	_ kanister.Func = (*kubeExecAllFunc)(nil)
)

const (
	KubeExecAllNamespaceArg      = "namespace"
	KubeExecAllPodsNameArg       = "pods"
	KubeExecAllContainersNameArg = "containers"
	KubeExecAllCommandArg        = "command"
)

type kubeExecAllFunc struct{}

func (*kubeExecAllFunc) Name() string {
	return "KubeExecAll"
}

func (*kubeExecAllFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) error {
	cli, err := kube.NewClient()
	if err != nil {
		return err
	}
	var namespace string
	var cmd, containers, pods []string
	if err = Arg(args, KubeExecAllNamespaceArg, &namespace); err != nil {
		return err
	}
	if err = Arg(args, KubeExecAllPodsNameArg, &pods); err != nil {
		return err
	}
	if err = Arg(args, KubeExecAllContainersNameArg, &containers); err != nil {
		return err
	}
	if err = Arg(args, KubeExecAllCommandArg, &cmd); err != nil {
		return err
	}
	return execAll(cli, namespace, pods, containers, cmd)
}

func (*kubeExecAllFunc) RequiredArgs() []string {
	return []string{KubeExecAllNamespaceArg, KubeExecAllPodsNameArg, KubeExecAllContainersNameArg, KubeExecAllCommandArg}
}

func execAll(cli kubernetes.Interface, namespace string, ps []string, cs []string, cmd []string) error {
	numContainers := len(ps) * len(cs)
	errChan := make(chan error, numContainers)
	// Run the command
	for _, p := range ps {
		for _, c := range cs {
			go func(p string, c string) {
				stdout, stderr, err := kube.Exec(cli, namespace, p, c, cmd)
				formatAndLog(p, c, stdout)
				formatAndLog(p, c, stderr)
				errChan <- err
			}(p, c)
		}
	}
	errs := make([]string, 0, numContainers)
	for i := 0; i < numContainers; i++ {
		err := <-errChan
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) != 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

func formatAndLog(podName string, containerName string, output string) {
	if output != "" {
		logs := regexp.MustCompile("[\r\n]").Split(output, -1)
		for _, l := range logs {
			if strings.TrimSpace(l) != "" {
				log.Info("Pod: ", podName, " Container: ", containerName, " Out: ", l)
			}
		}
	}
}
