package function

import (
	"context"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	ScaleDeploymentNamespaceArg = "namespace"
	ScaleDeploymentAppNameArg   = "name"
	ScaleDeploymentReplicas     = "replicas"
)

func init() {
	kanister.Register(&scaleDeploymentFunc{})
}

var (
	_ kanister.Func = (*scaleDeploymentFunc)(nil)
)

type scaleDeploymentFunc struct{}

func (*scaleDeploymentFunc) Name() string {
	return "ScaleDeployment"
}

func (*scaleDeploymentFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) error {
	var namespace, appName string
	var replicas int32
	err := Arg(args, ScaleSSNamespaceArg, &namespace)
	if err != nil {
		return err
	}
	err = Arg(args, ScaleSSAppNameArg, &appName)
	if err != nil {
		return err
	}
	err = Arg(args, ScaleSSReplicas, &replicas)
	if err != nil {
		return err
	}
	cli := kube.NewClient()
	return kube.ScaleDeployment(ctx, cli, namespace, appName, replicas)
}

func (*scaleDeploymentFunc) RequiredArgs() []string {
	return []string{ScaleDeploymentNamespaceArg, ScaleDeploymentAppNameArg, ScaleDeploymentReplicas}
}
