package function

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/kanisterio/kanister/pkg/blockstorage/awsebs"
	"github.com/kanisterio/kanister/pkg/blockstorage/getter"
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
	SnapshotDoesNotExistError        = "does not exist"
)

type deleteVolumeSnapshotFunc struct{}

func (*deleteVolumeSnapshotFunc) Name() string {
	return "DeleteVolumeSnapshot"
}

func deleteVolumeSnapshot(ctx context.Context, cli kubernetes.Interface, namespace, snapshotinfo string, profile *param.Profile, getter getter.Getter) (map[string]blockstorage.Provider, error) {
	PVCData := []VolumeSnapshotInfo{}
	err := json.Unmarshal([]byte(snapshotinfo), &PVCData)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not decode JSON data")
	}
	// providerList required for unit testing
	providerList := make(map[string]blockstorage.Provider)
	for _, pvcInfo := range PVCData {
		config := make(map[string]string)
		switch pvcInfo.Type {
		case blockstorage.TypeEBS:
			if err = ValidateProfile(profile); err != nil {
				return nil, errors.Wrap(err, "Profile validation failed")
			}
			config[awsebs.ConfigRegion] = pvcInfo.Region
			config[awsebs.AccessKeyID] = profile.Credential.KeyPair.ID
			config[awsebs.SecretAccessKey] = profile.Credential.KeyPair.Secret
		}
		provider, err := getter.Get(pvcInfo.Type, config)
		if err != nil {
			return nil, errors.Wrapf(err, "Could not get storage provider")
		}
		snapshot, err := provider.SnapshotGet(ctx, pvcInfo.SnapshotID)
		if err != nil {
			if strings.Contains(err.Error(), SnapshotDoesNotExistError) {
				log.Debugf("Snapshot %s already deleted", pvcInfo.SnapshotID)
			} else {
				return nil, errors.Wrapf(err, "Failed to get Snapshot from Provider")
			}
		}
		if err = provider.SnapshotDelete(ctx, snapshot); err != nil {
			return nil, err
		}
		log.Infof("Successfully deleted snapshot  %s", pvcInfo.SnapshotID)
		providerList[pvcInfo.PVCName] = provider
	}
	return providerList, nil
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
	_, err = deleteVolumeSnapshot(ctx, cli, namespace, snapshotinfo, tp.Profile, getter.New())
	return nil, err
}

func (*deleteVolumeSnapshotFunc) RequiredArgs() []string {
	return []string{DeleteVolumeSnapshotNamespaceArg, DeleteVolumeSnapshotManifestArg}
}
