package function

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/restic"
)

const (
	kanisterToolsImage                         = "kanisterio/kanister-tools:0.16.0"
	copyVolumeDataMountPoint                   = "/mnt/vol_data/%s"
	copyVolumeDataJobPrefix                    = "copy-vol-data-"
	CopyVolumeDataNamespaceArg                 = "namespace"
	CopyVolumeDataVolumeArg                    = "volume"
	CopyVolumeDataArtifactPrefixArg            = "dataArtifactPrefix"
	CopyVolumeDataOutputBackupID               = "backupID"
	CopyVolumeDataOutputBackupRoot             = "backupRoot"
	CopyVolumeDataOutputBackupArtifactLocation = "backupArtifactLocation"
	CopyVolumeDataEncryptionKeyArg             = "encryptionKey"
)

func init() {
	kanister.Register(&copyVolumeDataFunc{})
}

var _ kanister.Func = (*copyVolumeDataFunc)(nil)

type copyVolumeDataFunc struct{}

func (*copyVolumeDataFunc) Name() string {
	return "CopyVolumeData"
}

func copyVolumeData(ctx context.Context, cli kubernetes.Interface, tp param.TemplateParams, namespace, pvc, targetPath, encryptionKey string) (map[string]interface{}, error) {
	// Validate PVC exists
	if _, err := cli.CoreV1().PersistentVolumeClaims(namespace).Get(pvc, metav1.GetOptions{}); err != nil {
		return nil, errors.Wrapf(err, "Failed to retrieve PVC. Namespace %s, Name %s", namespace, pvc)
	}
	// Create a pod with PVCs attached
	mountPoint := fmt.Sprintf(copyVolumeDataMountPoint, pvc)
	pod, err := kube.CreatePod(ctx, cli, &kube.PodOptions{
		Namespace:    namespace,
		GenerateName: copyVolumeDataJobPrefix,
		Image:        kanisterToolsImage,
		Command:      []string{"sh", "-c", "tail -f /dev/null"},
		Volumes:      map[string]string{pvc: mountPoint},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create pod to copy volume data")
	}
	defer kube.DeletePod(context.Background(), cli, pod)

	// Wait for pod to reach running state
	if err := kube.WaitForPodReady(ctx, cli, pod.Namespace, pod.Name); err != nil {
		return nil, errors.Wrapf(err, "Failed while waiting for Pod %s to be ready", pod.Name)
	}

	// Get restic repository
	if err = restic.GetOrCreateRepository(cli, namespace, pod.Name, pod.Spec.Containers[0].Name, targetPath, encryptionKey, tp.Profile); err != nil {
		return nil, err
	}

	// Copy data to object store
	backupIdentifier := rand.String(10)
	cmd := restic.BackupCommand(tp.Profile, targetPath, backupIdentifier, mountPoint, encryptionKey)
	stdout, stderr, err := kube.Exec(cli, namespace, pod.Name, pod.Spec.Containers[0].Name, cmd)
	format.Log(pod.Name, pod.Spec.Containers[0].Name, stdout)
	format.Log(pod.Name, pod.Spec.Containers[0].Name, stderr)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create and upload backup")
	}
	return map[string]interface{}{
			CopyVolumeDataOutputBackupID:               backupIdentifier,
			CopyVolumeDataOutputBackupRoot:             mountPoint,
			CopyVolumeDataOutputBackupArtifactLocation: targetPath,
		},
		nil
}

func (*copyVolumeDataFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var namespace, vol, targetPath, encryptionKey string
	var err error
	if err = Arg(args, CopyVolumeDataNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, CopyVolumeDataVolumeArg, &vol); err != nil {
		return nil, err
	}
	if err = Arg(args, CopyVolumeDataArtifactPrefixArg, &targetPath); err != nil {
		return nil, err
	}
	if err = OptArg(args, CopyVolumeDataEncryptionKeyArg, &encryptionKey, restic.GeneratePassword()); err != nil {
		return nil, err
	}
	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	return copyVolumeData(ctx, cli, tp, namespace, vol, targetPath, encryptionKey)
}

func (*copyVolumeDataFunc) RequiredArgs() []string {
	return []string{CopyVolumeDataNamespaceArg, CopyVolumeDataVolumeArg, CopyVolumeDataArtifactPrefixArg}
}
