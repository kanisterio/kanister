package function

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/objectstore"
	"github.com/kanisterio/kanister/pkg/param"
)

func init() {
	kanister.Register(&createVolumeFromSnapshotFunc{})
}

var (
	_ kanister.Func = (*createVolumeFromSnapshotFunc)(nil)
)

const (
	CreateVolumeFromSnapshotNamespaceArg = "namespace"
	CreateVolumeFromSnapshotPathArg      = "snapshots"
)

type createVolumeFromSnapshotFunc struct{}

func (*createVolumeFromSnapshotFunc) Name() string {
	return "CreateVolumeFromSnapshot"
}

func createVolumeFromSnapshot(ctx context.Context, cli kubernetes.Interface, namespace, snapshotPath string, profile *param.Profile) error {
	data, err := objectstore.GetData(ctx, profile, objectstore.ProviderTypeS3, profile.Location.S3Compliant.Bucket, snapshotPath, "manifest.txt")
	if err != nil {
		return err
	}
	PVCData := []VolumeSnapshotInfo{}
	err = json.Unmarshal(data, &PVCData)
	if err != nil {
		return errors.Wrapf(err, "Could not decode JSON data")
	}
	for _, pvcInfo := range PVCData {
		var storageType string
		switch pvcInfo.StorageType {
		// TODO: use constants once blockstorage is moved to kanister repo
		case "EBS":
			storageType = "EBS"
		case "GPD":
			storageType = "GPD"
		case "AD":
			storageType = "AD"
		case "Cinder":
			storageType = "Cinder"
		case "Ceph":
			storageType = "Ceph"
		default:
			return errors.Errorf("Storage type %s not supported!", pvcInfo.StorageType)
		}
		log.Infof("snapshotId: %s, StorageType: %s, region: %s", pvcInfo.SnapshotID, storageType, pvcInfo.Region)
		if err := createPVCFromSnapshot(); err != nil {
			return errors.Wrapf(err, "Could not create PVC")
		}
	}
	return nil
}

func createPVCFromSnapshot() error {
	return errors.Wrapf(createPV(), "Could not create PV")
}

func createPV() error {
	return nil
}

func (kef *createVolumeFromSnapshotFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	var namespace, snapshots string
	if err = Arg(args, CreateVolumeFromSnapshotNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, CreateVolumeFromSnapshotPathArg, &snapshots); err != nil {
		return nil, err
	}
	return nil, createVolumeFromSnapshot(ctx, cli, namespace, snapshots, tp.Profile)
}

func (*createVolumeFromSnapshotFunc) RequiredArgs() []string {
	return []string{CreateVolumeFromSnapshotNamespaceArg, CreateVolumeFromSnapshotPathArg}
}
