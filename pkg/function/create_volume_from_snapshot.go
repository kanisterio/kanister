package function

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/blockstorage"
	"github.com/kanisterio/kanister/pkg/blockstorage/getter"
	awsconfig "github.com/kanisterio/kanister/pkg/config/aws"
	"github.com/kanisterio/kanister/pkg/kube"
	kubevolume "github.com/kanisterio/kanister/pkg/kube/volume"
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
	CreateVolumeFromSnapshotManifestArg  = "snapshots"
	CreateVolumeFromSnapshotPVCNamesArg  = "pvcNames"
)

type createVolumeFromSnapshotFunc struct{}

func (*createVolumeFromSnapshotFunc) Name() string {
	return "CreateVolumeFromSnapshot"
}

func createVolumeFromSnapshot(ctx context.Context, cli kubernetes.Interface, namespace, snapshotinfo string, pvcNames []string, profile *param.Profile, getter getter.Getter) (map[string]blockstorage.Provider, error) {
	PVCData := []VolumeSnapshotInfo{}
	err := json.Unmarshal([]byte(snapshotinfo), &PVCData)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not decode JSON data")
	}
	if len(pvcNames) > 0 && len(pvcNames) != len(PVCData) {
		return nil, errors.New("Invalid number of PVC names provided")
	}
	// providerList required for unit testing
	providerList := make(map[string]blockstorage.Provider)
	for i, pvcInfo := range PVCData {
		pvcName := pvcInfo.PVCName
		if len(pvcNames) > 0 {
			pvcName = pvcNames[i]
		}
		config := make(map[string]string)
		if err = ValidateProfile(profile, pvcInfo.Type); err != nil {
			return nil, errors.Wrap(err, "Profile validation failed")
		}
		switch pvcInfo.Type {
		case blockstorage.TypeEBS:
			config[awsconfig.ConfigRegion] = pvcInfo.Region
			config[awsconfig.AccessKeyID] = profile.Credential.KeyPair.ID
			config[awsconfig.SecretAccessKey] = profile.Credential.KeyPair.Secret
		case blockstorage.TypeGPD:
			config[blockstorage.GoogleProjectID] = profile.Credential.KeyPair.ID
			config[blockstorage.GoogleServiceKey] = profile.Credential.KeyPair.Secret
		}
		provider, err := getter.Get(pvcInfo.Type, config)
		if err != nil {
			return nil, errors.Wrapf(err, "Could not get storage provider %v", pvcInfo.Type)
		}
		_, err = cli.CoreV1().PersistentVolumeClaims(namespace).Get(pvcName, metav1.GetOptions{})
		if err == nil {
			if err = kubevolume.DeletePVC(cli, namespace, pvcName); err != nil {
				return nil, err
			}
		}
		snapshot, err := provider.SnapshotGet(ctx, pvcInfo.SnapshotID)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to get Snapshot from Provider")
		}

		tags := map[string]string{
			"pvcname": pvcName,
		}
		snapshot.Volume.VolumeType = pvcInfo.VolumeType
		snapshot.Volume.Az = pvcInfo.Az
		snapshot.Volume.Tags = pvcInfo.Tags
		vol, err := provider.VolumeCreateFromSnapshot(ctx, *snapshot, tags)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to create volume from snapshot, snapID: %s", snapshot.ID)
		}

		annotations := map[string]string{}
		pvc, err := kubevolume.CreatePVC(ctx, cli, namespace, pvcName, vol.Size, vol.ID, annotations)
		if err != nil {
			return nil, errors.Wrapf(err, "Unable to create PVC for volume %v", *vol)
		}
		pv, err := kubevolume.CreatePV(ctx, cli, vol, vol.Type, annotations)
		if err != nil {
			return nil, errors.Wrapf(err, "Unable to create PV for volume %v", *vol)
		}
		log.Infof("Restore/Create volume from snapshot completed for pvc: %s, volume: %s", pvc, pv)
		providerList[pvcInfo.PVCName] = provider
	}
	return providerList, nil
}

func (kef *createVolumeFromSnapshotFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	var namespace, snapshotinfo string
	var pvcNames []string
	if err = Arg(args, CreateVolumeFromSnapshotNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, CreateVolumeFromSnapshotManifestArg, &snapshotinfo); err != nil {
		return nil, err
	}
	if err = OptArg(args, CreateVolumeFromSnapshotPVCNamesArg, &pvcNames, nil); err != nil {
		return nil, err
	}
	_, err = createVolumeFromSnapshot(ctx, cli, namespace, snapshotinfo, pvcNames, tp.Profile, getter.New())
	return nil, err
}

func (*createVolumeFromSnapshotFunc) RequiredArgs() []string {
	return []string{CreateVolumeFromSnapshotNamespaceArg, CreateVolumeFromSnapshotManifestArg}
}
