package function

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/format"
	kankopia "github.com/kanisterio/kanister/pkg/kopia"
	kopiacmd "github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	RestoreDataUsingKopiaServerFuncName = "RestoreDataUsingKopiaServer"
	// SparseRestoreOption is the key for specifiying whether to do a sparse restore
	SparseRestoreOption = "sparseRestore"
)

type restoreDataUsingKopiaServerFunc struct{}

func init() {
	err := kanister.Register(&restoreDataUsingKopiaServerFunc{})
	if err != nil {
		return
	}
}

var _ kanister.Func = (*restoreDataUsingKopiaServerFunc)(nil)

func (*restoreDataUsingKopiaServerFunc) Name() string {
	return RestoreDataUsingKopiaServerFuncName
}

func (*restoreDataUsingKopiaServerFunc) RequiredArgs() []string {
	return []string{
		RestoreDataBackupIdentifierArg,
		RestoreDataImageArg,
		RestoreDataNamespaceArg,
		RestoreDataRestorePathArg,
		kankopia.KopiaUserPassphraseArg,
		kankopia.KopiaTLSCertSecretDataArg,
	}
}

func (*restoreDataUsingKopiaServerFunc) Arguments() []string {
	return []string{
		RestoreDataBackupIdentifierArg,
		RestoreDataImageArg,
		RestoreDataNamespaceArg,
		RestoreDataRestorePathArg,
		kankopia.KopiaUserPassphraseArg,
		kankopia.KopiaTLSCertSecretDataArg,
		RestoreDataPodArg,
		RestoreDataVolsArg,
	}
}

func (*restoreDataUsingKopiaServerFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]any) (map[string]any, error) {
	var (
		err            error
		image          string
		namespace      string
		restorePath    string
		snapID         string
		userPassphrase string
		cert           string
	)
	if err = Arg(args, RestoreDataBackupIdentifierArg, &snapID); err != nil {
		return nil, err
	}
	if err = Arg(args, RestoreDataImageArg, &image); err != nil {
		return nil, err
	}
	if err = Arg(args, RestoreDataNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, RestoreDataRestorePathArg, &restorePath); err != nil {
		return nil, err
	}
	if err = Arg(args, kankopia.KopiaUserPassphraseArg, &userPassphrase); err != nil {
		return nil, err
	}

	if err = Arg(args, kankopia.KopiaTLSCertSecretDataArg, &cert); err != nil {
		return nil, err
	}

	fingerprint, err := kankopia.ExtractFingerprintFromCertificateJSON(cert)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch Kopia API Server Certificate Secret Data from blueprint")
	}

	// Validate and get optional arguments
	pod, vols, err := validateAndGetOptArgsForRestore(args)
	if err != nil {
		return nil, err
	}

	if len(vols) == 0 {
		// Fetch Volumes
		vols, err = FetchPodVolumes(pod, tp)
		if err != nil {
			return nil, err
		}
	}

	username := tp.RepositoryServer.Username
	hostname, userAccessPassphrase, err := getHostNameAndUserPassPhraseFromRepoServer(userPassphrase)

	if err != nil {
		return nil, errors.Wrap(err, "Failed to get hostname/user passphrase from Options")
	}

	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create Kubernetes client")
	}

	serverAddress, err := getRepositoryServerAddress(cli, *tp.RepositoryServer)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get the Kopia Repository Server Address")
	}

	_, sparseRestore := tp.Options[SparseRestoreOption]

	return restoreDataFromServer(
		ctx,
		cli,
		hostname,
		image,
		restoreDataJobPrefix,
		namespace,
		restorePath,
		serverAddress,
		fingerprint,
		snapID,
		username,
		userAccessPassphrase,
		sparseRestore,
		vols,
	)
}

func restoreDataFromServer(
	ctx context.Context,
	cli kubernetes.Interface,
	hostname,
	image,
	jobPrefix,
	namespace,
	restorePath,
	serverAddress,
	fingerprint,
	snapID,
	username,
	userPassphrase string,
	sparseRestore bool,
	vols map[string]string,
) (map[string]any, error) {
	// Validate volumes
	for pvc := range vols {
		if _, err := cli.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, pvc, metav1.GetOptions{}); err != nil {
			return nil, errors.Wrap(err, "Failed to retrieve PVC from namespace: "+namespace+" name: "+pvc)
		}
	}

	options := &kube.PodOptions{
		Namespace:    namespace,
		GenerateName: jobPrefix,
		Image:        image,
		Command:      []string{"bash", "-c", "tail -f /dev/null"},
		Volumes:      vols,
	}

	kankopia.SetLabelsToPodOptionsIfRequired(options)
	kankopia.SetAnnotationsToPodOptionsIfRequired(options)
	kankopia.SetResourceRequirementsToPodOptionsIfRequired(options)
	pr := kube.NewPodRunner(cli, options)
	podFunc := restoreDataFromServerPodFunc(
		cli,
		hostname,
		namespace,
		restorePath,
		serverAddress,
		fingerprint,
		snapID,
		username,
		userPassphrase,
		sparseRestore,
	)
	return pr.Run(ctx, podFunc)
}

func restoreDataFromServerPodFunc(
	cli kubernetes.Interface,
	hostname,
	namespace,
	restorePath,
	serverAddress,
	fingerprint,
	snapID,
	username,
	userPassphrase string,
	sparseRestore bool,
) func(ctx context.Context, pod *corev1.Pod) (map[string]any, error) {
	return func(ctx context.Context, pod *corev1.Pod) (map[string]any, error) {
		// Wait for pod to reach running state
		if err := kube.WaitForPodReady(ctx, cli, pod.Namespace, pod.Name); err != nil {
			return nil, errors.Wrap(err, "Failed while waiting for Pod: "+pod.Name+" to be ready")
		}

		contentCacheMB, metadataCacheMB := kopiacmd.GetCacheSizeSettingsForSnapshot()
		configFile, logDirectory := kankopia.GetCustomConfigFileAndLogDirectory(hostname)

		cmd := kopiacmd.RepositoryConnectServerCommand(
			kopiacmd.RepositoryServerCommandArgs{
				UserPassword:    userPassphrase,
				ConfigFilePath:  configFile,
				LogDirectory:    logDirectory,
				CacheDirectory:  kopiacmd.DefaultCacheDirectory,
				Hostname:        hostname,
				ServerURL:       serverAddress,
				Fingerprint:     fingerprint,
				Username:        username,
				ContentCacheMB:  contentCacheMB,
				MetadataCacheMB: metadataCacheMB,
			})
		stdout, stderr, err := kube.Exec(cli, namespace, pod.Name, pod.Spec.Containers[0].Name, cmd, nil)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stdout)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stderr)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to connect to Kopia API server")
		}

		cmd = kopiacmd.SnapshotRestore(
			kopiacmd.SnapshotRestoreCommandArgs{
				CommandArgs: &kopiacmd.CommandArgs{
					RepoPassword:   "",
					ConfigFilePath: configFile,
					LogDirectory:   logDirectory,
				},
				SnapID:                 snapID,
				TargetPath:             restorePath,
				SparseRestore:          sparseRestore,
				IgnorePermissionErrors: true,
			})
		stdout, stderr, err = kube.Exec(cli, namespace, pod.Name, pod.Spec.Containers[0].Name, cmd, nil)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stdout)
		format.Log(pod.Name, pod.Spec.Containers[0].Name, stderr)
		return nil, errors.Wrap(err, "Failed to restore backup from Kopia API server")
	}
}

func validateAndGetOptArgsForRestore(args map[string]any) (pod string, vols map[string]string, err error) {
	if err = OptArg(args, RestoreDataPodArg, &pod, ""); err != nil {
		return pod, vols, err
	}
	if err = OptArg(args, RestoreDataVolsArg, &vols, nil); err != nil {
		return pod, vols, err
	}
	if (pod != "") == (len(vols) > 0) {
		return pod, vols, errors.New(fmt.Sprintf("Require exactly one of %s or %s", RestoreDataPodArg, RestoreDataVolsArg))
	}
	return pod, vols, nil
}
