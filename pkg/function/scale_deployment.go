package function

import (
	"context"
	"strconv"

	"github.com/pkg/errors"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/kube"
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

func (*scaleDeploymentFunc) Exec(ctx context.Context, args ...string) error {
	if len(args) != 3 {
		return errors.Errorf("ScaleDeployment requires 3 arguments. Got: %#v", args)
	}
	cli := kube.NewClient()
	namespace, appName := args[0], args[1]
	scaleNumber, err := strconv.Atoi(args[2])
	if err != nil {
		return errors.Wrapf(err, "Failed to convert string arg %s to int.", args[2])
	}
	return kube.ScaleDeployment(ctx, cli, namespace, appName, int32(scaleNumber))
}
