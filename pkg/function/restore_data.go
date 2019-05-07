package function

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/restic"
)

const (
	restoreDataJobPrefix = "restore-data-"
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
	// RestoreDataBackupTagArg provides a unique tag added to the backup artifacts
	RestoreDataBackupTagArg = "backupTag"
)

func init() {
	kanister.Register(&restoreDataFunc{})
}

var _ kanister.Func = (*restoreDataFunc)(nil)

type restoreDataFunc struct{}

func (*restoreDataFunc) Name() string {
	return "RestoreData"
}

func validateAndGetOptArgs(args map[string]interface{}) (string, string, string, map[string]string, string, string, error) {
	var restorePath, encryptionKey, pod, tag, id string
	var vols map[string]string
	var err error

	if err = OptArg(args, RestoreDataRestorePathArg, &restorePath, "/"); err != nil {
		return restorePath, encryptionKey, pod, vols, tag, id, err
	}
	if err = OptArg(args, RestoreDataEncryptionKeyArg, &encryptionKey, restic.GeneratePassword()); err != nil {
		return restorePath, encryptionKey, pod, vols, tag, id, err
	}
	if err = OptArg(args, RestoreDataPodArg, &pod, ""); err != nil {
		return restorePath, encryptionKey, pod, vols, tag, id, err
	}
	if err = OptArg(args, RestoreDataVolsArg, &vols, nil); err != nil {
		return restorePath, encryptionKey, pod, vols, tag, id, err
	}
	if (pod != "") == (len(vols) > 0) {
		return restorePath, encryptionKey, pod, vols, tag, id,
			errors.Errorf("Require one argument: %s or %s", RestoreDataPodArg, RestoreDataVolsArg)
	}
	if err = OptArg(args, RestoreDataBackupTagArg, &tag, nil); err != nil {
		return restorePath, encryptionKey, pod, vols, tag, id, err
	}
	if err = OptArg(args, RestoreDataBackupIdentifierArg, &id, nil); err != nil {
		return restorePath, encryptionKey, pod, vols, tag, id, err
	}
	if (tag != "") == (id != "") {
		return restorePath, encryptionKey, pod, vols, tag, id,
			errors.Errorf("Require one argument: %s or %s", RestoreDataBackupTagArg, RestoreDataBackupIdentifierArg)
	}
	return restorePath, encryptionKey, pod, vols, tag, id, nil
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

func restoreData(ctx context.Context, cli kubernetes.Interface, tp param.TemplateParams, namespace, encryptionKey, backupArtifactPrefix, restorePath, backupTag, backupID string, vols map[string]string) (map[string]interface{}, error) {
	// Validate volumes
	for pvc := range vols {
		if _, err := cli.CoreV1().PersistentVolumeClaims(namespace).Get(pvc, metav1.GetOptions{}); err != nil {
			return nil, errors.Wrapf(err, "Failed to retrieve PVC. Namespace %s, Name %s", namespace, pvc)
		}
	}
	options := &kube.PodOptions{
		Namespace:    namespace,
		GenerateName: restoreDataJobPrefix,
		Image:        kanisterToolsImage,
		Command:      []string{"sh", "-c", "tail -f /dev/null"},
		Volumes:      vols,
	}
	pr := kube.NewPodRunner(cli, options)
	podFunc := restoreDataPodFunc(cli, tp, namespace, encryptionKey, backupArtifactPrefix, restorePath, backupTag, backupID)
	return pr.Run(ctx, podFunc)
}

func restoreDataPodFunc(cli kubernetes.Interface, tp param.TemplateParams, namespace, encryptionKey, backupArtifactPrefix, restorePath, backupTag, backupID string) func(ctx context.Context, pod *v1.Pod) (map[string]interface{}, error) {
	return func(ctx context.Context, pod *v1.Pod) (map[string]interface{}, error) {
		// Wait for pod to reach running state
		if err := kube.WaitForPodReady(ctx, cli, pod.Namespace, pod.Name); err != nil {
			return nil, errors.Wrapf(err, "Failed while waiting for Pod %s to be ready", pod.Name)
		}
		pw, err := getPodWriter(cli, ctx, pod.Namespace, pod.Name, pod.Spec.Containers[0].Name, tp.Profile)
		if err != nil {
			return nil, err
		}
		defer cleanUpCredsFile(ctx, pw, pod.Namespace, pod.Name, pod.Spec.Containers[0].Name)
		var cmd []string
		// Generate restore command based on the identifier passed
		if backupTag != "" {
			cmd = restic.RestoreCommandByTag(tp.Profile, backupArtifactPrefix, backupTag, restorePath, encryptionKey)
		} else if backupID != "" {
			cmd = restic.RestoreCommandByID(tp.Profile, backupArtifactPrefix, backupID, restorePath, encryptionKey)
		}
		stdout, stderr, err := kube.Exec(cli, namespace, pod.Name, pod.Spec.Containers[0].Name, cmd, nil)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stdout)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stderr)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to create and upload backup")
		}
		out, err := parseLogAndCreateOutput(stdout)
		return out, errors.Wrap(err, "Failed to parse phase output")
	}
}

func (*restoreDataFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var namespace, image, backupArtifactPrefix, backupTag, backupID string
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
	// Validate and get optional arguments
	restorePath, encryptionKey, pod, vols, backupTag, backupID, err := validateAndGetOptArgs(args)
	if err != nil {
		return nil, err
	}
	// Validate profile
	if err = validateProfile(tp.Profile); err != nil {
		return nil, err
	}
	if len(vols) == 0 {
		// Fetch Volumes
		vols, err = fetchPodVolumes(pod, tp)
		if err != nil {
			return nil, err
		}
	}
	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	return restoreData(ctx, cli, tp, namespace, encryptionKey, backupArtifactPrefix, restorePath, backupTag, backupID, vols)
}

func (*restoreDataFunc) RequiredArgs() []string {
	return []string{RestoreDataNamespaceArg, RestoreDataImageArg,
		RestoreDataBackupArtifactPrefixArg}
}
