package function

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/kanisterio/errkit"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/ephemeral"
	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/secrets"
	"github.com/kanisterio/kanister/pkg/validate"
)

const (
	// FunctionOutputVersion returns version
	FunctionOutputVersion = "version"

	// since pod labels and annotations argument are going to be named the
	// same for all the kanister functions that support these arguments, instead
	// of creating these for the functions, it's better to have a const here.
	PodLabelsArg      = "podLabels"
	PodAnnotationsArg = "podAnnotations"
)

const (
	defaultContainerAnn = "kubectl.kubernetes.io/default-container"
)

// ExecAndLog executes a command using the provided executor and logs the output.
// It encapsulates the common pattern of executing a command and logging both stdout and stderr.
// Returns the stdout and stderr content as strings along with any execution error.
func ExecAndLog(ctx context.Context, executor kube.PodCommandExecutor, cmd []string, pod *corev1.Pod) (stdout, stderr string, err error) {
	var stdoutBuf, stderrBuf bytes.Buffer

	err = executor.Exec(ctx, cmd, nil, &stdoutBuf, &stderrBuf)

	// Get the container name - use the first container as that's the pattern used in existing code
	containerName := pod.Spec.Containers[0].Name

	// Log the output regardless of whether the command succeeded or failed
	format.LogWithCtx(ctx, pod.Name, containerName, stdoutBuf.String())
	format.LogWithCtx(ctx, pod.Name, containerName, stderrBuf.String())

	return stdoutBuf.String(), stderrBuf.String(), err
}

// ExecAndLogNoCtx executes a command using the provided executor and logs the output without context.
// It encapsulates the common pattern of executing a command and logging both stdout and stderr.
// Returns the stdout and stderr content as strings along with any execution error.
func ExecAndLogNoCtx(ctx context.Context, executor kube.PodCommandExecutor, cmd []string, pod *corev1.Pod) (stdout, stderr string, err error) {
	var stdoutBuf, stderrBuf bytes.Buffer

	err = executor.Exec(ctx, cmd, nil, &stdoutBuf, &stderrBuf)

	// Get the container name - use the first container as that's the pattern used in existing code
	containerName := pod.Spec.Containers[0].Name

	// Log the output regardless of whether the command succeeded or failed
	format.Log(pod.Name, containerName, stdoutBuf.String())
	format.Log(pod.Name, containerName, stderrBuf.String())

	return stdoutBuf.String(), stderrBuf.String(), err
}

// KubeExecAndLog executes a command using kube.Exec and logs the output with context.
// It encapsulates the common pattern of executing a command via kube.Exec and logging both stdout and stderr.
// Returns the stdout and stderr content as strings along with any execution error.
func KubeExecAndLog(ctx context.Context, cli kubernetes.Interface, namespace, pod, container string, cmd []string, stdin io.Reader) (stdout, stderr string, err error) {
	stdout, stderr, err = kube.Exec(ctx, cli, namespace, pod, container, cmd, stdin)

	// Log the output regardless of whether the command succeeded or failed
	format.LogWithCtx(ctx, pod, container, stdout)
	format.LogWithCtx(ctx, pod, container, stderr)

	return stdout, stderr, err
}

// KubeExecAndLogNoCtx executes a command using kube.Exec and logs the output without context.
// It encapsulates the common pattern of executing a command via kube.Exec and logging both stdout and stderr.
// Returns the stdout and stderr content as strings along with any execution error.
func KubeExecAndLogNoCtx(ctx context.Context, cli kubernetes.Interface, namespace, pod, container string, cmd []string, stdin io.Reader) (stdout, stderr string, err error) {
	stdout, stderr, err = kube.Exec(ctx, cli, namespace, pod, container, cmd, stdin)

	// Log the output regardless of whether the command succeeded or failed
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)

	return stdout, stderr, err
}

// ValidateCredentials verifies if the given credentials have appropriate values set
func ValidateCredentials(creds *param.Credential) error {
	if creds == nil {
		return errkit.New("Empty credentials")
	}
	switch creds.Type {
	case param.CredentialTypeKeyPair:
		if creds.KeyPair == nil {
			return errkit.New("Empty KeyPair field")
		}
		if len(creds.KeyPair.ID) == 0 {
			return errkit.New("Access key ID is not set")
		}
		if len(creds.KeyPair.Secret) == 0 {
			return errkit.New("Secret access key is not set")
		}
		return nil
	case param.CredentialTypeSecret:
		return secrets.ValidateCredentials(creds.Secret)
	case param.CredentialTypeKopia:
		if creds.KopiaServerSecret == nil {
			return errkit.New("Empty KopiaServerSecret field")
		}
		if len(creds.KopiaServerSecret.Username) == 0 {
			return errkit.New("Kopia Username is not set")
		}
		if len(creds.KopiaServerSecret.Password) == 0 {
			return errkit.New("Kopia UserPassphrase is not set")
		}
		if len(creds.KopiaServerSecret.Hostname) == 0 {
			return errkit.New("Kopia Hostname is not set")
		}
		if len(creds.KopiaServerSecret.Cert) == 0 {
			return errkit.New("Kopia TLSCert is not set")
		}
		return nil
	default:
		return errkit.New(fmt.Sprintf("Unsupported type '%s' for credentials", creds.Type))
	}
}

// ValidateProfile verifies if the given profile has valid creds and location type
func ValidateProfile(profile *param.Profile) error {
	if profile == nil {
		return errkit.New("Profile must be non-nil")
	}
	if err := ValidateCredentials(&profile.Credential); err != nil {
		return err
	}
	switch profile.Location.Type {
	case crv1alpha1.LocationTypeS3Compliant:
	case crv1alpha1.LocationTypeGCS:
	case crv1alpha1.LocationTypeAzure:
	case crv1alpha1.LocationTypeKopia:
	default:
		return errkit.New("Location type not supported")
	}
	return nil
}

type nopRemover struct {
}

var _ kube.PodFileRemover = nopRemover{}

func (nr nopRemover) Remove(ctx context.Context) error {
	return nil
}

func (nr nopRemover) Path() string {
	return ""
}

// MaybeWriteProfileCredentials creates a file with Google credentials if the given profile points to a GCS location, otherwise does nothing
func MaybeWriteProfileCredentials(ctx context.Context, pc kube.PodController, profile *param.Profile) (kube.PodFileRemover, error) {
	if profile.Location.Type == crv1alpha1.LocationTypeGCS {
		pfw, err := pc.GetFileWriter()
		if err != nil {
			return nil, errkit.Wrap(err, "Unable to write Google credentials")
		}

		remover, err := pfw.Write(ctx, consts.GoogleCloudCredsFilePath, bytes.NewBufferString(profile.Credential.KeyPair.Secret))
		if err != nil {
			return nil, errkit.Wrap(err, "Unable to write Google credentials")
		}

		return remover, nil
	}

	return nopRemover{}, nil
}

// GetPodWriter creates a file with Google credentials if the given profile points to a GCS location
//
//nolint:revive // context-as-argument: maintaining backward compatibility for public API
func GetPodWriter(cli kubernetes.Interface, ctx context.Context, namespace, podName, containerName string, profile *param.Profile) (kube.PodWriter, error) {
	if profile.Location.Type == crv1alpha1.LocationTypeGCS {
		pw := kube.NewPodWriter(cli, consts.GoogleCloudCredsFilePath, bytes.NewBufferString(profile.Credential.KeyPair.Secret))
		if err := pw.Write(ctx, namespace, podName, containerName); err != nil {
			return nil, err
		}
		return pw, nil
	}
	return nil, nil
}

// CleanUpCredsFile is used to remove the file created by the given PodWriter
func CleanUpCredsFile(ctx context.Context, pw kube.PodWriter, namespace, podName, containerName string) {
	if pw != nil {
		if err := pw.Remove(ctx, namespace, podName, containerName); err != nil {
			log.Error().WithContext(ctx).Print("Could not delete the temp file")
		}
	}
}

// FetchPodVolumes returns a map of PVCName->MountPath for a given pod
func FetchPodVolumes(pod string, tp param.TemplateParams) (map[string]string, error) {
	switch {
	case tp.Deployment != nil:
		if pvcToMountPath, ok := tp.Deployment.PersistentVolumeClaims[pod]; ok {
			return pvcToMountPath, nil
		}
		return nil, errkit.New("Failed to find volumes for the Pod: " + pod)
	case tp.StatefulSet != nil:
		if pvcToMountPath, ok := tp.StatefulSet.PersistentVolumeClaims[pod]; ok {
			return pvcToMountPath, nil
		}
		return nil, errkit.New("Failed to find volumes for the Pod: " + pod)
	default:
		return nil, errkit.New("Invalid Template Params")
	}
}

// ResolveArtifactPrefix appends the bucket name as a suffix to the given artifact path if not already present
func ResolveArtifactPrefix(artifactPrefix string, profile *param.Profile) string {
	ps := strings.Split(artifactPrefix, "/")
	if ps[0] == profile.Location.Bucket {
		return artifactPrefix
	}
	return path.Join(profile.Location.Bucket, artifactPrefix)
}

func createPostgresSecret(cli kubernetes.Interface, name, namespace, username, password string) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data: map[string][]byte{
			"username": []byte(username),
			"password": []byte(password),
		},
	}
	_, err := cli.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	return err
}

func deletePostgresSecret(cli kubernetes.Interface, name, namespace string) error {
	return cli.CoreV1().Secrets(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
}

func isAuroraCluster(engine string) bool {
	for _, v := range []string{string(DBEngineAurora), string(DBEngineAuroraMySQL), string(DBEngineAuroraPostgreSQL)} {
		if engine == v {
			return true
		}
	}
	return false
}

// ValidatePodLabelsAndAnnotations validates the labels and annotations that are
// passed to a Kanister function (`funcName`) using `podLabels` and `podAnnotations` args.
func ValidatePodLabelsAndAnnotations(funcName string, args map[string]any) error {
	labels, err := PodLabelsFromFunctionArgs(args)
	if err != nil {
		return errkit.Wrap(err, "Kanister function validation failed, while getting pod labels from function args", "funcName", funcName)
	}

	if err = validate.ValidateLabels(labels); err != nil {
		return errkit.Wrap(err, "Kanister function validation failed, while validating labels", "funcName", funcName)
	}

	annotations, err := PodAnnotationsFromFunctionArgs(args)
	if err != nil {
		return errkit.Wrap(err, "Kanister function validation failed, while getting pod annotations from function args", "funcName", funcName)
	}
	if err = validate.ValidateAnnotations(annotations); err != nil {
		return errkit.Wrap(err, "Kanister function validation failed, while validating annotations", "funcName", funcName)
	}
	return nil
}

func PodLabelsFromFunctionArgs(args map[string]any) (map[string]string, error) {
	for k, v := range args {
		if k == PodLabelsArg && v != nil {
			labels, ok := v.(map[string]interface{})
			if !ok {
				return nil, errkit.New("podLabels are not in correct format. Expected format is map[string]interface{}.")
			}
			return mapStringInterfaceToString(labels), nil
		}
	}
	return nil, nil
}

// mapStringInterfaceToString accepts a map of `string` and `interface{}` and creates
// a map of `string` and `string` from passed map and returns that.
// If a value in the passed map is not of type `string`, it will be skipped.
func mapStringInterfaceToString(m map[string]interface{}) map[string]string {
	res := map[string]string{}
	for k, v := range m {
		switch v := v.(type) {
		case string:
			res[k] = v
		default:
			log.Info().Print("Map value is not of type string, while converting map[string]interface{} to map[string]string. Skipping.", map[string]interface{}{"value": v})
		}
	}
	return res
}

func PodAnnotationsFromFunctionArgs(args map[string]any) (map[string]string, error) {
	for k, v := range args {
		if k == PodAnnotationsArg && v != nil {
			annotations, ok := v.(map[string]interface{})
			if !ok {
				return nil, errkit.New("podAnnotations are not in correct format. expected format is map[string]string.")
			}
			return mapStringInterfaceToString(annotations), nil
		}
	}
	return nil, nil
}

type ActionSetAnnotations map[string]string

// MergeBPAnnotations merges the annotations provided in the blueprint with the annotations
// configured via actionset. If the same key is present at both places, the one in blueprint
// will be used.
func (a ActionSetAnnotations) MergeBPAnnotations(bpAnnotations map[string]string) map[string]string {
	annotations := map[string]string{}
	for k, v := range a {
		annotations[k] = v
	}
	for k, v := range bpAnnotations {
		annotations[k] = v
	}

	return annotations
}

type ActionSetLabels map[string]string

// MergeBPLabels merges the labels provided in the blueprint with the labels
// configured via actionset. If the same key is present at both places, the one in blueprint
// will be used.
func (a ActionSetLabels) MergeBPLabels(bpLabels map[string]string) map[string]string {
	labels := map[string]string{}
	for k, v := range a {
		labels[k] = v
	}
	for k, v := range bpLabels {
		labels[k] = v
	}

	return labels
}

func PrepareAndRunPod(
	ctx context.Context,
	cli kubernetes.Interface,
	namespace, jobPrefix, image string,
	command []string,
	vols map[string]string,
	podOverride crv1alpha1.JSONMap,
	annotations, labels map[string]string,
	podFunc func(ctx context.Context, pc kube.PodController) (map[string]interface{}, error),
) (map[string]any, error) {
	// Validate volumes
	validatedVols := make(map[string]kube.VolumeMountOptions)
	for pvcName, mountPoint := range vols {
		pvc, err := cli.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, pvcName, metav1.GetOptions{})
		if err != nil {
			return nil, errkit.Wrap(err, "Failed to retrieve PVC.", "namespace", namespace, "name", pvcName)
		}

		validatedVols[pvcName] = kube.VolumeMountOptions{
			MountPath: mountPoint,
			ReadOnly:  kube.PVCContainsReadOnlyAccessMode(pvc),
		}
	}

	// Create PodOptions
	options := &kube.PodOptions{
		Namespace:    namespace,
		GenerateName: jobPrefix,
		Image:        image,
		Command:      command,
		Volumes:      validatedVols,
		PodOverride:  podOverride,
		Annotations:  annotations,
		Labels:       labels,
	}

	// Apply ephemeral pod changes
	if err := ephemeral.PodOptions.Apply(options); err != nil {
		return nil, errkit.Wrap(err, "Failed to apply ephemeral pod options")
	}

	// Create and run the pod
	pr := kube.NewPodRunner(cli, options)
	return pr.Run(ctx, podFunc)
}
