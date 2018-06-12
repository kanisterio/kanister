package function

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
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
	// BackupDataBackupArtifactArg provides the path to store artifacts on the object store
	BackupDataBackupArtifactArg = "backupArtifact"
)

func init() {
	kanister.Register(&backupDataFunc{})
}

var _ kanister.Func = (*backupDataFunc)(nil)

type backupDataFunc struct{}

func (*backupDataFunc) Name() string {
	return "BackupData"
}

func generateBackupCommand(includePath, destArtifact string, profile *param.Profile) []string {
	// Command to export credentials
	cmd := []string{"set", "+o", "xtrace"}
	cmd = append(cmd, "export", fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", profile.Credential.KeyPair.Secret))
	cmd = append(cmd, "export", fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", profile.Credential.KeyPair.ID))
	cmd = append(cmd, "set", "-o", "xtrace")
	// Command to tar and compress
	cmd = append(cmd, "tar", "-cf", "-", "-C", includePath, ".", "|", "gzip", "-", "|")
	// Command to dump on the object store
	cmd = append(cmd, "aws")
	if profile.Location.S3Compliant.Endpoint != "" {
		cmd = append(cmd, "--endpoint", profile.Location.S3Compliant.Endpoint)
	}
	if profile.SkipSSLVerify {
		cmd = append(cmd, "--no-verify-ssl")
	}
	cmd = append(cmd, "s3", "cp", "-", destArtifact)
	command := strings.Join(cmd, " ")
	return []string{"bash", "-o", "errexit", "-o", "pipefail", "-c", command}
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

func (*backupDataFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) error {
	var namespace, pod, container, includePath, backupArtifact string
	var err error
	if err = Arg(args, BackupDataNamespaceArg, &namespace); err != nil {
		return err
	}
	if err = Arg(args, BackupDataPodArg, &pod); err != nil {
		return err
	}
	if err = Arg(args, BackupDataContainerArg, &container); err != nil {
		return err
	}
	if err = Arg(args, BackupDataIncludePathArg, &includePath); err != nil {
		return err
	}
	if err = Arg(args, BackupDataBackupArtifactArg, &backupArtifact); err != nil {
		return err
	}
	// Validate the Profile
	if err = validateProfile(tp.Profile); err != nil {
		return errors.Wrapf(err, "Failed to validate Profile")
	}
	cli, err := kube.NewClient()
	if err != nil {
		return errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	// Create backup and dump it on the object store
	cmd := generateBackupCommand(includePath, backupArtifact, tp.Profile)
	stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd)
	formatAndLog(pod, container, stdout)
	formatAndLog(pod, container, stderr)
	if err != nil {
		return errors.Wrapf(err, "Failed to create and upload backup")
	}
	return nil
}

func (*backupDataFunc) RequiredArgs() []string {
	return []string{BackupDataNamespaceArg, BackupDataPodArg, BackupDataContainerArg,
		BackupDataIncludePathArg, BackupDataBackupArtifactArg}
}
