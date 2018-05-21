package function

import (
	"context"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	ScaleSSNamespaceArg = "namespace"
	ScaleSSAppNameArg   = "name"
	ScaleSSReplicas     = "replicas"
)

func init() {
	kanister.Register(&scaleStatefulSetFunc{})
}

var (
	_ kanister.Func = (*scaleStatefulSetFunc)(nil)
)

type scaleStatefulSetFunc struct{}

func (*scaleStatefulSetFunc) Name() string {
	return "ScaleStatefulSet"
}

func (*scaleStatefulSetFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) error {
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
	return kube.ScaleStatefulSet(ctx, cli, namespace, appName, replicas)
}

func (*scaleStatefulSetFunc) RequiredArgs() []string {
	return []string{ScaleSSNamespaceArg, ScaleSSAppNameArg, ScaleSSReplicas}
}
