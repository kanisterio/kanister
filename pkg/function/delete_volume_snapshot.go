package function

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
)

func init() {
	kanister.Register(&deleteVolumeSnapshotFunc{})
}

var (
	_ kanister.Func = (*deleteVolumeSnapshotFunc)(nil)
)

const (
	DeleteVolumeSnapshotNamespaceArg = "namespace"
	DeleteVolumeSnapshotManifestArg  = "snapshots"
)

type deleteVolumeSnapshotFunc struct{}

func (*deleteVolumeSnapshotFunc) Name() string {
	return "DeleteVolumeSnapshot"
}

func deleteVolumeSnapshot(ctx context.Context, cli kubernetes.Interface, namespace, snapshotinfo string, profile *param.Profile) error {
	return nil
}

func (kef *deleteVolumeSnapshotFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	var namespace, snapshotinfo string
	if err = Arg(args, DeleteVolumeSnapshotNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, DeleteVolumeSnapshotManifestArg, &snapshotinfo); err != nil {
		return nil, err
	}
	return nil, deleteVolumeSnapshot(ctx, cli, namespace, snapshotinfo, tp.Profile)
}

func (*deleteVolumeSnapshotFunc) RequiredArgs() []string {
	return []string{DeleteVolumeSnapshotNamespaceArg, DeleteVolumeSnapshotManifestArg}
}
