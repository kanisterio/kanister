package function

import (
	"context"
	"strconv"

	"github.com/pkg/errors"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/kube"
)

func init() {
	kanister.Register(&scaleStatefulSetFunc{})
}

var _ kanister.Func = (*scaleStatefulSetFunc)(nil)

type scaleStatefulSetFunc struct{}

func (*scaleStatefulSetFunc) Name() string {
	return "ScaleStatefulSet"
}

func (*scaleStatefulSetFunc) Exec(ctx context.Context, args ...string) error {
	cli := kube.NewClient()
	if len(args) != 3 {
		return errors.Errorf("Incorrect number of arguments. Expected 3. Got: %#v", args)
	}
	namespace, name, r := args[0], args[1], args[2]
	replicas, err := strconv.Atoi(r)
	if err != nil {
		return err
	}
	return kube.ScaleStatefulSet(ctx, cli, namespace, name, int32(replicas))
}
