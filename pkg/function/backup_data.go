package function

import (
	"context"
	"regexp"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/rand"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/format"
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
	// BackupDataEncryptionKeyArg provides the encryption key to be used for backups
	BackupDataEncryptionKeyArg = "encryptionKey"
	// BackupDataOutputBackupID is the key used for returning backup ID output
	BackupDataOutputBackupID = "backupID"
	// BackupDataOutputBackupTag is the key used for returning backupTag output
	BackupDataOutputBackupTag = "backupTag"
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

func getSnapshotIDFromLog(output string) string {
	if output == "" {
		return ""
	}
	logs := regexp.MustCompile("[\n]").Split(output, -1)
	for _, l := range logs {
		// Log should contain "snapshot ABC123 saved"
		pattern := regexp.MustCompile(`snapshot\s(.*?)\ssaved$`)
		match := pattern.FindAllStringSubmatch(l, 1)
		if match != nil {
			return match[0][1]
		}
	}
	return ""
}

func (*backupDataFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var namespace, pod, container, includePath, backupArtifactPrefix, encryptionKey string
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
	if err = OptArg(args, BackupDataEncryptionKeyArg, &encryptionKey, restic.GeneratePassword()); err != nil {
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

	if err = restic.GetOrCreateRepository(cli, namespace, pod, container, backupArtifactPrefix, encryptionKey, tp.Profile); err != nil {
		return nil, err
	}

	// Create backup and dump it on the object store
	backupTag := rand.String(10)
	cmd := restic.BackupCommandByTag(tp.Profile, backupArtifactPrefix, backupTag, includePath, encryptionKey)
	stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create and upload backup")
	}
	// Get the snapshot ID from log
	backupID := getSnapshotIDFromLog(stdout)
	if backupID == "" {
		return nil, errors.New("Failed to parse the backup ID from logs")
	}
	output := map[string]interface{}{
		BackupDataOutputBackupID:  backupID,
		BackupDataOutputBackupTag: backupTag,
	}
	return output, nil
}

func (*backupDataFunc) RequiredArgs() []string {
	return []string{BackupDataNamespaceArg, BackupDataPodArg, BackupDataContainerArg,
		BackupDataIncludePathArg, BackupDataBackupArtifactPrefixArg}
}
