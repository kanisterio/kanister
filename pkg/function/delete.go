package function

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pkg/errors"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/kube"
)

func init() {
	kanister.Register(&deleteGeneratedFunc{})
}

var _ kanister.Func = (*deleteGeneratedFunc)(nil)

type deleteGeneratedFunc struct{}

func (*deleteGeneratedFunc) Name() string {
	return "DeleteGeneratedResources"
}

func (*deleteGeneratedFunc) Exec(ctx context.Context, args ...string) error {
	cli := kube.NewClient()
	if len(args) != 4 {
		return errors.Errorf("Incorrect number of arguments. Expected 4. Got: %#v", args)
	}
	resource, namespace, generateName, size := args[0], args[1], args[2], args[3]
	n, err := strconv.Atoi(size)
	if err != nil {
		return err
	}
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("%s%d", generateName, i)
		switch resource {
		case "pvc":
			err = cli.Core().PersistentVolumeClaims(namespace).Delete(name, nil)
			if err != nil {
				break
			}
		default:
			err = errors.Errorf("Delete not implemented for resource %s", resource)
			break
		}
	}
	return err
}
