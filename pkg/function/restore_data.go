package function

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	// RestoreDataNamespaceArg provides the namespace
	RestoreDataNamespaceArg = "namespace"
	// RestoreDataPodArg provides the pod connected to the data volume
	RestoreDataPodArg = "pod"
	// RestoreDataImageArg provides the image of the container with required tools
	RestoreDataImageArg = "image"
	// RestoreDataBackupArtifactArg provides the path of the backed up artifact
	RestoreDataBackupArtifactArg = "backupArtifact"
	// RestoreDataRestorePathArg provides the path to restore backed up data
	RestoreDataRestorePathArg = "restorePath"
)

func init() {
	kanister.Register(&restoreDataFunc{})
}

var _ kanister.Func = (*restoreDataFunc)(nil)

type restoreDataFunc struct{}

func (*restoreDataFunc) Name() string {
	return "RestoreData"
}

func fetchPodVolumes(pod string, tp param.TemplateParams) (map[string]string, error) {
	switch {
	case tp.Deployment != nil:
		if pvcToMountPath, ok := tp.Deployment.PersistentVolumeClaims[pod]; ok {
			return pvcToMountPath, nil
		}
		return nil, errors.New("Failed to find volumes for the Pod: " + pod)
	case tp.StatefulSet != nil:
		for i, p := range tp.StatefulSet.Pods {
			if p != pod {
				continue
			}
			if len(tp.StatefulSet.PersistentVolumeClaims) > i {
				return tp.StatefulSet.PersistentVolumeClaims[i], nil
			}
		}
		return nil, errors.New("Failed to find volumes for the Pod: " + pod)
	default:
		return nil, errors.New("Invalid Template Params")
	}
}

func generateRestoreCommand(backupArtifact, restorePath string, profile *param.Profile) []string {
	// Command to export credentials
	cmd := []string{"export", fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s\n", profile.Credential.KeyPair.Secret)}
	cmd = append(cmd, "export", fmt.Sprintf("AWS_ACCESS_KEY_ID=%s\n", profile.Credential.KeyPair.ID))
	// Command to retrieve from object store
	cmd = append(cmd, "aws")
	if profile.Location.S3Compliant.Endpoint != "" {
		cmd = append(cmd, "--endpoint", profile.Location.S3Compliant.Endpoint)
	}
	if profile.SkipSSLVerify {
		cmd = append(cmd, "--no-verify-ssl")
	}
	cmd = append(cmd, "s3", "cp", backupArtifact, "-", "|")
	// Command to extract
	cmd = append(cmd, "gunzip", "-c", "-", "|", "tar", "-xf", "-", "-C", restorePath)
	command := strings.Join(cmd, " ")
	return []string{"bash", "-o", "errexit", "-o", "pipefail", "-c", command}
}

func (*restoreDataFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) error {
	var namespace, pod, image, backupArtifact, restorePath string
	var err error
	if err = Arg(args, RestoreDataNamespaceArg, &namespace); err != nil {
		return err
	}
	if err = Arg(args, RestoreDataPodArg, &pod); err != nil {
		return err
	}
	if err = Arg(args, RestoreDataImageArg, &image); err != nil {
		return err
	}
	if err = Arg(args, RestoreDataBackupArtifactArg, &backupArtifact); err != nil {
		return err
	}
	if err = Arg(args, RestoreDataRestorePathArg, &restorePath); err != nil {
		return err
	}
	// Validate profile
	err = validateProfile(tp.Profile)
	if err != nil {
		return err
	}
	// Generate restore command
	cmd := generateRestoreCommand(backupArtifact, restorePath, tp.Profile)
	// Fetch Volumes
	vols, err := fetchPodVolumes(pod, tp)
	if err != nil {
		return err
	}
	// Call PrepareData with generated command
	cli, err := kube.NewClient()
	if err != nil {
		return errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	return prepareData(ctx, cli, namespace, "", image, vols, cmd...)
}

func (*restoreDataFunc) RequiredArgs() []string {
	return []string{RestoreDataNamespaceArg, RestoreDataPodArg, RestoreDataImageArg,
		RestoreDataBackupArtifactArg, RestoreDataRestorePathArg}
}
