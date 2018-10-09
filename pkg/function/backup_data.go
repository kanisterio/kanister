package function

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/restic"
)

const (
	// BackupDataNamespaceArg provides the namespace
	BackupDataNamespaceArg = "namespace"
	// BackupDataPodArg provides the pod connected to the data volume
	BackupDataPodArg = "pod"
	// BackupDataContainerArg provides the container on which the backup is taken
	BackupDataContainerArg = "container"
	// BackupDataIncludePathArg provides the path of the volume or sub-path for required backup
	BackupDataIncludePathArg = "includePath"
	// BackupDataBackupArtifactPrefixArg provides the path to store artifacts on the object store
	BackupDataBackupArtifactPrefixArg = "backupArtifactPrefix"
	// BackupDataBackupIdentifierArg provides a unique ID added to the artifacts
	BackupDataBackupIdentifierArg = "backupIdentifier"
)

func init() {
	kanister.Register(&backupDataFunc{})
}

var _ kanister.Func = (*backupDataFunc)(nil)

type backupDataFunc struct{}

func (*backupDataFunc) Name() string {
	return "BackupData"
}

func validateProfile(profile *param.Profile) error {
	if profile == nil {
		return errors.New("Profile must be non-nil")
	}
	if profile.Location.Type != crv1alpha1.LocationTypeS3Compliant {
		return errors.New("Location type not supported")
	}
	return nil
}

func getOrCreateRepository(cli kubernetes.Interface, namespace, pod, container, artifactPrefix string, profile *param.Profile) error {
	// Use the snapshots command to check if the repository exists
	cmd := restic.SnapshotsCommand(profile, artifactPrefix)
	stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd)
	formatAndLog(pod, container, stdout)
	formatAndLog(pod, container, stderr)
	if err != nil {
		// Create a repository
		cmd := restic.InitCommand(profile, artifactPrefix)
		stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd)
		formatAndLog(pod, container, stdout)
		formatAndLog(pod, container, stderr)
		if err != nil {
			return errors.Wrapf(err, "Failed to create object store backup location")
		}
	}
	return nil
}

func (*backupDataFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {

	var namespace, pod, container, includePath, backupArtifactPrefix, backupIdentifier string
	var err error
	if err = Arg(args, BackupDataNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, BackupDataPodArg, &pod); err != nil {
		return nil, err
	}
	if err = Arg(args, BackupDataContainerArg, &container); err != nil {
		return nil, err
	}
	if err = Arg(args, BackupDataIncludePathArg, &includePath); err != nil {
		return nil, err
	}
	if err = Arg(args, BackupDataBackupArtifactPrefixArg, &backupArtifactPrefix); err != nil {
		return nil, err
	}
	if err = Arg(args, BackupDataBackupIdentifierArg, &backupIdentifier); err != nil {
		return nil, err
	}
	// Validate the Profile
	if err = validateProfile(tp.Profile); err != nil {
		return nil, errors.Wrapf(err, "Failed to validate Profile")
	}
	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}

	if err = getOrCreateRepository(cli, namespace, pod, container, backupArtifactPrefix, tp.Profile); err != nil {
		return nil, err
	}

	// Create backup and dump it on the object store
	cmd := restic.BackupCommand(tp.Profile, backupArtifactPrefix, backupIdentifier, includePath)
	stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd)
	formatAndLog(pod, container, stdout)
	formatAndLog(pod, container, stderr)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create and upload backup")
	}
	return nil, nil
}

func (*backupDataFunc) RequiredArgs() []string {
	return []string{BackupDataNamespaceArg, BackupDataPodArg, BackupDataContainerArg,
		BackupDataIncludePathArg, BackupDataBackupArtifactPrefixArg, BackupDataBackupIdentifierArg}
}
