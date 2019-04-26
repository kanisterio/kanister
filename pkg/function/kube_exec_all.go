package function

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/format"
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

func (*kubeExecAllFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	cli, err := kube.NewClient()
	if err != nil {
		return nil, err
	}
	var namespace, pods, containers string
	var cmd []string
	if err = Arg(args, KubeExecAllNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, KubeExecAllPodsNameArg, &pods); err != nil {
		return nil, err
	}
	if err = Arg(args, KubeExecAllContainersNameArg, &containers); err != nil {
		return nil, err
	}
	if err = Arg(args, KubeExecAllCommandArg, &cmd); err != nil {
		return nil, err
	}
	ps := strings.Fields(pods)
	cs := strings.Fields(containers)
	return execAll(cli, namespace, ps, cs, cmd)
}

func (*kubeExecAllFunc) RequiredArgs() []string {
	return []string{KubeExecAllNamespaceArg, KubeExecAllPodsNameArg, KubeExecAllContainersNameArg, KubeExecAllCommandArg}
}

func execAll(cli kubernetes.Interface, namespace string, ps []string, cs []string, cmd []string) (map[string]interface{}, error) {
	numContainers := len(ps) * len(cs)
	errChan := make(chan error, numContainers)
	output := ""
	// Run the command
	for _, p := range ps {
		for _, c := range cs {
			go func(p string, c string) {
				stdout, stderr, err := kube.Exec(cli, namespace, p, c, cmd, nil)
				format.Log(p, c, stdout)
				format.Log(p, c, stderr)
				errChan <- err
				output = output + "\n" + stdout
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
		return nil, errors.New(strings.Join(errs, "\n"))
	}
	out, err := parseLogAndCreateOutput(output)
	if err != nil {
		return nil, err
	}
	return out, nil
}
