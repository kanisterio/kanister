package function

import (
	"context"
	"io"

	api "github.com/kanisterio/datamover/api/v1alpha1"
	"github.com/kanisterio/datamover/pkg/client"
	"github.com/kanisterio/errkit"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/field"
	kopiacmd "github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	DMArgNamespace        = "namespace"
	DMArgImage            = "image"
	DMArgDatamoverSession = "datamoverSession"
	DMArgClientSecret     = "clientSecret" // TODO: clientSecretVolume???
	DMArgConfig           = "config"
	DMArgSecrets          = "secrets"
	DMArgEnv              = "env"
	DMArgPodOptions       = "podOptions"
)

type DataMoverSessionRef struct {
	Namespace string
	Name      string
}

type datamoverArgs struct {
	namespace     string
	image         string
	sessionRef    DataMoverSessionRef
	clientSecret  string
	config        *string
	configSecrets map[string]string
	env           []corev1.EnvVar
	podOptions    api.PodOptions
}

func getDatamoverArgs(tp param.TemplateParams, args map[string]interface{}) (*datamoverArgs, error) {
	dmArgs := datamoverArgs{}

	if err := Arg(args, DMArgNamespace, &dmArgs.namespace); err != nil {
		return nil, err
	}
	if err := Arg(args, DMArgImage, &dmArgs.image); err != nil {
		return nil, err
	}

	var sessionRef DataMoverSessionRef
	if err := Arg(args, DMArgDatamoverSession, &sessionRef); err != nil {
		return nil, err
	}
	dmArgs.sessionRef = sessionRef

	// FIXME: configmap from actionset
	var argConfigMap string
	if err := OptArg(args, DMArgConfig, &argConfigMap, ""); err != nil {
		return nil, err
	}
	if argConfigMap != "" {
		configMap, ok := tp.ConfigMaps[argConfigMap]
		if ok {
			dmArgs.config = &configMap.Name
		}
	}

	var actionSecrets []string
	if err := OptArg(args, DMArgSecrets, &actionSecrets, []string{}); err != nil {
		return nil, err
	}

	secretNames := map[string]string{}
	for _, actionSecret := range actionSecrets {
		secretSpec, ok := tp.Secrets[actionSecret]
		if ok {
			if secretSpec.Namespace == dmArgs.namespace {
				secretNames[actionSecret] = secretSpec.Name
			} else {
				log.Info().Print("Secret reference from different namespace. Ignoring", field.M{"secretName": secretSpec.Name, "secretNamespace": secretSpec.Namespace})
			}
		}
	}
	dmArgs.configSecrets = secretNames

	// TODO: we can validate that this secret is in datamover clients secret if we have access to datamover server secrets
	var actionClientSecretName string
	if err := Arg(args, DMArgClientSecret, &actionClientSecretName); err != nil {
		return nil, err
	}

	clientSecretSpec, ok := tp.Secrets[actionClientSecretName]
	if !ok {
		return nil, errkit.New("Client secret not found in the actionset:", "secretName", actionClientSecretName)
	}
	if clientSecretSpec.Namespace != dmArgs.namespace {
		return nil, errkit.New("Client secret in the actionset is in the wrong namespace:", "secretName", actionClientSecretName, "secretNamespace", clientSecretSpec.Namespace, "namespace", dmArgs.namespace)
	}
	dmArgs.clientSecret = clientSecretSpec.Name

	var env map[string]string
	if err := OptArg(args, DMArgEnv, &env, map[string]string{}); err != nil {
		return nil, err
	}
	dmArgs.env = []corev1.EnvVar{}
	for k, v := range env {
		dmArgs.env = append(dmArgs.env, corev1.EnvVar{Name: k, Value: v})
	}

	var podOptions *api.PodOptions
	if err := OptArg(args, DMArgPodOptions, &podOptions, &api.PodOptions{}); err != nil {
		return nil, err
	}

	podOverride := crv1alpha1.JSONMap{}
	if podOptions != nil && podOptions.PodOverride != nil {
		podOverride = crv1alpha1.JSONMap(podOptions.PodOverride)
	}
	podOverride, err := getPodOverride(podOverride, tp)
	if err != nil {
		return nil, err
	}
	podOptions.PodOverride = api.PodOverride(podOverride)

	dmArgs.podOptions = *podOptions

	return &dmArgs, nil
}

func getPodOverride(podOverride crv1alpha1.JSONMap, tp param.TemplateParams) (crv1alpha1.JSONMap, error) {
	actionSetOverride := tp.PodOverride
	var err error
	if actionSetOverride != nil {
		podOverride, err = kube.CreateAndMergeJSONPatch(podOverride, actionSetOverride)
		if err != nil {
			return nil, err
		}
	}
	return podOverride, nil
}

func monitorDatamoverPod(ctx context.Context, cli kubernetes.Interface, pod *corev1.Pod) error {
	pc, err := kube.NewPodControllerForExistingPod(cli, pod)
	if err != nil {
		return err
	}

	ctx = field.Context(ctx, consts.PodNameKey, pod.Name)
	ctx = field.Context(ctx, consts.ContainerNameKey, pod.Spec.Containers[0].Name)
	go func() {
		<-ctx.Done()
		err := pc.StopPod(context.Background(), kube.PodControllerInfiniteStopTime, int64(0))
		if err != nil {
			log.WithError(err).Print("Failed to delete pod", field.M{"PodName": pod.Name})
		}
	}()

	if err := pc.WaitForPodReady(ctx); err != nil {
		return errkit.Wrap(err, "Failed while waiting for Pod to be ready", "pod", pc.PodName())
	}

	// Wait for pod completion
	if err := pc.WaitForPodCompletion(ctx); err != nil {
		return errkit.Wrap(err, "Failed while waiting for Pod to complete", "pod", pc.PodName())
	}
	return nil
}

func streamPodLogs(ctx context.Context, cli kubernetes.Interface, pod corev1.Pod, containerName string) (io.ReadCloser, error) {
	ctx = field.Context(ctx, consts.LogKindKey, consts.LogKindDatapath)
	// Fetch logs from the pod
	// FIXME default value for injestor container name??
	reader, err := kube.StreamPodLogs(ctx, cli, pod.Namespace, pod.Name, containerName, nil)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to fetch logs from the pod")
	}
	return reader, nil
}

func parseKopiaOutput(reader io.ReadCloser) (map[string]interface{}, error) {
	podOutput, err := io.ReadAll(reader)
	if err != nil {
		return nil, errkit.Wrap(err, "Failed to read logs stream from the pod")
	}
	podOutputString := string(podOutput)

	log.Info().Print("Pod output", field.M{"PodOutput": podOutput})

	snapInfo, err := kopiacmd.ParseSnapshotCreateOutput(podOutputString, podOutputString)
	if err != nil {
		return nil, errkit.Wrap(err, "Cannot parse kopia snapshot create output")
	}

	log.Info().Print("Snapshot info", field.M{"info": snapInfo})

	// FIXME: this needs to be checked. Some inconsistency in stats
	var logSize, phySize, fileCount int64
	if snapInfo.Stats != nil {
		stats := snapInfo.Stats
		logSize = stats.SizeHashedB + stats.SizeCachedB
		phySize = stats.SizeUploadedB
		fileCount = stats.FilesHashed + stats.FilesCached
	}

	return map[string]any{
		CopyVolumeDataOutputRootID:          snapInfo.RootID,
		CopyVolumeDataOutputBackupID:        snapInfo.SnapshotID,
		CopyVolumeDataOutputBackupSize:      logSize,
		CopyVolumeDataOutputPhysicalSize:    phySize,
		CopyVolumeDataOutputBackupFileCount: fileCount,
	}, nil
}

func makeCreateClientArgs(args datamoverArgs, operation client.Operation) client.CreateClientArgs {
	return client.CreateClientArgs{
		Operation:        operation,
		Namespace:        args.namespace,
		Image:            args.image,
		SessionNamespace: args.sessionRef.Namespace,
		SessionName:      args.sessionRef.Name,
		ConfigMap:        args.config,
		Secrets:          args.configSecrets,
		Env:              args.env,
		PodOptions:       args.podOptions,
		CredentialsConfig: client.ClientCredentialsSecret{
			SecretName: args.clientSecret,
		},
	}
}
