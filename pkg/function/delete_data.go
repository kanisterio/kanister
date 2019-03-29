package function

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/restic"
)

const (
	// DeleteDataNamespaceArg provides the namespace
	DeleteDataNamespaceArg = "namespace"
	// DeleteDataBackupArtifactPrefixArg provides the path to restore backed up data
	DeleteDataBackupArtifactPrefixArg = "backupArtifactPrefix"
	// DeleteDataBackupIdentifierArg provides a unique ID added to the backed up artifacts
	DeleteDataBackupIdentifierArg = "backupID"
	// DeleteDataBackupTagArg provides a unique tag added to the backed up artifacts
	DeleteDataBackupTagArg = "backupTag"
	// DeleteDataEncryptionKeyArg provides the encryption key to be used for deletes
	DeleteDataEncryptionKeyArg = "encryptionKey"
	// DeleteDataReclaimSpace provides a way to specify if space should be reclaimed
	DeleteDataReclaimSpace = "reclaimSpace"
	deleteDataJobPrefix    = "delete-data-"
)

func init() {
	kanister.Register(&deleteDataFunc{})
}

var _ kanister.Func = (*deleteDataFunc)(nil)

type deleteDataFunc struct{}

func (*deleteDataFunc) Name() string {
	return "DeleteData"
}

func (*deleteDataFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var namespace, deleteArtifactPrefix, deleteIdentifier, deleteTag, encryptionKey string
	var reclaimSpace bool
	var err error
	if err = Arg(args, DeleteDataNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, DeleteDataBackupArtifactPrefixArg, &deleteArtifactPrefix); err != nil {
		return nil, err
	}
	if err = OptArg(args, DeleteDataBackupIdentifierArg, &deleteIdentifier, ""); err != nil {
		return nil, err
	}
	if err = OptArg(args, DeleteDataBackupTagArg, &deleteTag, ""); err != nil {
		return nil, err
	}
	if err = OptArg(args, DeleteDataEncryptionKeyArg, &encryptionKey, restic.GeneratePassword()); err != nil {
		return nil, err
	}
	if err = OptArg(args, DeleteDataReclaimSpace, &reclaimSpace, false); err != nil {
		return nil, err
	}
	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	pod, err := kube.CreatePod(ctx, cli, &kube.PodOptions{
		Namespace:    namespace,
		GenerateName: deleteDataJobPrefix,
		Image:        kanisterToolsImage,
		Command:      []string{"sh", "-c", "tail -f /dev/null"},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create pod to delete data")
	}
	defer kube.DeletePod(context.Background(), cli, pod)

	// Wait for pod to reach running state
	if err := kube.WaitForPodReady(ctx, cli, pod.Namespace, pod.Name); err != nil {
		return nil, errors.Wrapf(err, "Failed while waiting for Pod %s to be ready", pod.Name)
	}
	if (deleteIdentifier != "") == (deleteTag != "") {
		return nil, errors.Errorf("Require one argument: %s or %s", DeleteDataBackupIdentifierArg, DeleteDataBackupTagArg)
	}
	if deleteTag != "" {
		cmd := restic.SnapshotsCommandByTag(tp.Profile, deleteArtifactPrefix, deleteTag, encryptionKey)
		stdout, stderr, err := kube.Exec(cli, namespace, pod.Name, pod.Spec.Containers[0].Name, cmd)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stdout)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stderr)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to forget data, could not get snapshotID from tag, Tag: %s", deleteTag)
		}
		deleteIdentifier, err = GetSnapshotIDFromLog(stdout)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to forget data, could not get snapshotID from tag, Tag: %s", deleteTag)
		}
	}
	if deleteIdentifier != "" {
		cmd := restic.ForgetCommandByID(tp.Profile, deleteArtifactPrefix, deleteIdentifier, encryptionKey)
		stdout, stderr, err := kube.Exec(cli, namespace, pod.Name, pod.Spec.Containers[0].Name, cmd)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stdout)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stderr)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to forget data")
		}
	}

	if reclaimSpace {
		cmd := restic.PruneCommand(tp.Profile, deleteArtifactPrefix, encryptionKey)
		stdout, stderr, err := kube.Exec(cli, namespace, pod.Name, pod.Spec.Containers[0].Name, cmd)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stdout)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stderr)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to prune data after forget")
		}
	}
	return nil, nil
}

func (*deleteDataFunc) RequiredArgs() []string {
	return []string{DeleteDataNamespaceArg, DeleteDataBackupArtifactPrefixArg}
}

// GetSnapshotIDFromLog gets the SnapshotID from log
func GetSnapshotIDFromLog(output string) (string, error) {
	var result []map[string]interface{}
	err := json.Unmarshal([]byte(output), &result)
	if err != nil {
		return "", errors.WithMessage(err, "Failed to unmarshall output from snapshotCommand")
	}
	if len(result) != 1 {
		return "", errors.New("Snapshot not found")
	}
	snapId := result[0]["short_id"]
	return snapId.(string), nil
}
