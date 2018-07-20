package function

import (
	"context"
	"strings"

	"github.com/pkg/errors"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	ScaleWorkloadNamespaceArg = "namespace"
	ScaleWorkloadNameArg      = "name"
	ScaleWorkloadKindArg      = "kind"
	ScaleWorkloadReplicas     = "replicas"
)

func init() {
	kanister.Register(&scaleWorkloadFunc{})
}

var (
	_ kanister.Func = (*scaleWorkloadFunc)(nil)
)

type scaleWorkloadFunc struct{}

func (*scaleWorkloadFunc) Name() string {
	return "ScaleWorkload"
}

func (*scaleWorkloadFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) error {
	var namespace, kind, name string
	var replicas int32
	namespace, kind, name, replicas, err := getArgs(tp, args)
	if err != nil {
		return err
	}

	cli, err := kube.NewClient()
	if err != nil {
		return errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	switch strings.ToLower(kind) {
	case param.StatefulSetKind:
		return kube.ScaleStatefulSet(ctx, cli, namespace, name, replicas)
	case param.DeploymentKind:
		return kube.ScaleDeployment(ctx, cli, namespace, name, replicas)
	default:
		return errors.New("Workload type not supported " + kind)
	}
}

func (*scaleWorkloadFunc) RequiredArgs() []string {
	return []string{ScaleWorkloadReplicas}
}

func getArgs(tp param.TemplateParams, args map[string]interface{}) (namespace, kind, name string, replicas int32, err error) {
	err = Arg(args, ScaleWorkloadReplicas, &replicas)
	if err != nil {
		return namespace, kind, name, replicas, err
	}

	// Populate default values for optional arguments from template parameters
	switch {
	case tp.StatefulSet != nil:
		kind = param.StatefulSetKind
		name = tp.StatefulSet.Name
		namespace = tp.StatefulSet.Namespace
	case tp.Deployment != nil:
		kind = param.DeploymentKind
		name = tp.Deployment.Name
		namespace = tp.Deployment.Namespace
	default:
		if !ArgExists(args, ScaleWorkloadNamespaceArg) || !ArgExists(args, ScaleWorkloadNameArg) || !ArgExists(args, ScaleWorkloadKindArg) {
			return namespace, kind, name, replicas, errors.New("Workload information not available via defaults or namespace/name/kind parameters")
		}
	}

	err = OptArg(args, ScaleWorkloadNamespaceArg, &namespace, namespace)
	if err != nil {
		return namespace, kind, name, replicas, err
	}
	err = OptArg(args, ScaleWorkloadNameArg, &name, name)
	if err != nil {
		return namespace, kind, name, replicas, err
	}
	err = OptArg(args, ScaleWorkloadKindArg, &kind, kind)
	if err != nil {
		return namespace, kind, name, replicas, err
	}
	return namespace, kind, name, replicas, err
}
