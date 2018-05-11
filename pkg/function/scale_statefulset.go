package function

import (
	"context"
	"strconv"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/pkg/errors"
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

func (*scaleStatefulSetFunc) Exec(ctx context.Context, args ...string) error {
	if len(args) != 3 {
		return errors.Errorf("ScaleStatefulSet requires 3 arguments. Got: %#v", args)
	}
	cli := kube.NewClient()
	namespace, appName := args[0], args[1]
	scaleNumber, err := strconv.Atoi(args[2])
	if err != nil {
		return errors.Wrapf(err, "Failed to convert string arg %s to int.", args[2])
	}
	return kube.ScaleStatefulSet(ctx, cli, namespace, appName, int32(scaleNumber))
}
