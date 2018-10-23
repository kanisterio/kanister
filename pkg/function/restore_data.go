package function

import (
	"context"

	"github.com/pkg/errors"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/restic"
)

const (
	// RestoreDataNamespaceArg provides the namespace
	RestoreDataNamespaceArg = "namespace"
	// RestoreDataImageArg provides the image of the container with required tools
	RestoreDataImageArg = "image"
	// RestoreDataBackupArtifactPrefixArg provides the path of the backed up artifact
	RestoreDataBackupArtifactPrefixArg = "backupArtifactPrefix"
	// RestoreDataRestorePathArg provides the path to restore backed up data
	RestoreDataRestorePathArg = "restorePath"
	// RestoreDataBackupIdentifierArg provides a unique ID added to the backed up artifacts
	RestoreDataBackupIdentifierArg = "backupIdentifier"
	// RestoreDataPodArg provides the pod connected to the data volume
	RestoreDataPodArg = "pod"
	// RestoreDataVolsArg provides a map of PVC->mountPaths to be attached
	RestoreDataVolsArg = "volumes"
	// RestoreDataEncryptionKeyArg provides the encryption key used during backup
	RestoreDataEncryptionKeyArg = "encryptionKey"
)

func init() {
	kanister.Register(&restoreDataFunc{})
}

var _ kanister.Func = (*restoreDataFunc)(nil)

type restoreDataFunc struct{}

func (*restoreDataFunc) Name() string {
	return "RestoreData"
}

func validateOptArgs(pod string, vols map[string]string) error {
	if (pod != "") != (len(vols) > 0) {
		return nil
	}
	return errors.Errorf("Require one argument: %s or %s", RestoreDataPodArg, RestoreDataVolsArg)
}

func fetchPodVolumes(pod string, tp param.TemplateParams) (map[string]string, error) {
	switch {
	case tp.Deployment != nil:
		if pvcToMountPath, ok := tp.Deployment.PersistentVolumeClaims[pod]; ok {
			return pvcToMountPath, nil
		}
		return nil, errors.New("Failed to find volumes for the Pod: " + pod)
	case tp.StatefulSet != nil:
		if pvcToMountPath, ok := tp.StatefulSet.PersistentVolumeClaims[pod]; ok {
			return pvcToMountPath, nil
		}
		return nil, errors.New("Failed to find volumes for the Pod: " + pod)
	default:
		return nil, errors.New("Invalid Template Params")
	}
}

func (*restoreDataFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var namespace, pod, image, backupArtifactPrefix, restorePath, backupIdentifier, encryptionKey string
	var vols map[string]string
	var err error
	if err = Arg(args, RestoreDataNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, RestoreDataImageArg, &image); err != nil {
		return nil, err
	}
	if err = Arg(args, RestoreDataBackupArtifactPrefixArg, &backupArtifactPrefix); err != nil {
		return nil, err
	}
	if err = Arg(args, RestoreDataBackupIdentifierArg, &backupIdentifier); err != nil {
		return nil, err
	}
	if err = OptArg(args, RestoreDataRestorePathArg, &restorePath, "/"); err != nil {
		return nil, err
	}
	if err = OptArg(args, RestoreDataPodArg, &pod, ""); err != nil {
		return nil, err
	}
	if err = OptArg(args, RestoreDataVolsArg, &vols, nil); err != nil {
		return nil, err
	}
	if err = OptArg(args, RestoreDataEncryptionKeyArg, &encryptionKey, restic.GeneratePassword()); err != nil {
		return nil, err
	}
	if err = validateOptArgs(pod, vols); err != nil {
		return nil, err
	}
	// Validate profile
	if err = validateProfile(tp.Profile); err != nil {
		return nil, err
	}
	// Generate restore command
	cmd := restic.RestoreCommand(tp.Profile, backupArtifactPrefix, backupIdentifier, restorePath, encryptionKey)
	if len(vols) == 0 {
		// Fetch Volumes
		vols, err = fetchPodVolumes(pod, tp)
		if err != nil {
			return nil, err
		}
	}
	// Call PrepareData with generated command
	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	return nil, prepareData(ctx, cli, namespace, "", image, vols, cmd...)
}

func (*restoreDataFunc) RequiredArgs() []string {
	return []string{RestoreDataNamespaceArg, RestoreDataImageArg,
		RestoreDataBackupArtifactPrefixArg, RestoreDataBackupIdentifierArg}
}
