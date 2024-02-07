package function

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/restic"
)

const (
	// CheckRepositoryFuncName gives the name of the function
	CheckRepositoryFuncName = "CheckRepository"
	// CheckRepositoryArtifactPrefixArg provides the path to restore backed up data
	CheckRepositoryArtifactPrefixArg = "backupArtifactPrefix"
	// CheckRepositoryEncryptionKeyArg provides the encryption key to be used for deletes
	CheckRepositoryEncryptionKeyArg = "encryptionKey"
	// CheckRepositoryPodOverrideArg contains pod specs to override default pod specs
	CheckRepositoryPodOverrideArg    = "podOverride"
	CheckRepositoryJobPrefix         = "check-repository-"
	CheckRepositoryPasswordIncorrect = "passwordIncorrect"
	CheckRepositoryRepoDoesNotExist  = "repoUnavailable"
)

func init() {
	_ = kanister.Register(&CheckRepositoryFunc{})
}

var _ kanister.Func = (*CheckRepositoryFunc)(nil)

type CheckRepositoryFunc struct {
	progressPercent string
}

func (*CheckRepositoryFunc) Name() string {
	return CheckRepositoryFuncName
}

func CheckRepository(ctx context.Context, cli kubernetes.Interface, tp param.TemplateParams, encryptionKey, targetPaths, jobPrefix string, insecureTLS bool, podOverride crv1alpha1.JSONMap) (map[string]interface{}, error) {
	namespace, err := kube.GetControllerNamespace()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get controller namespace")
	}
	options := &kube.PodOptions{
		Namespace:    namespace,
		GenerateName: jobPrefix,
		Image:        consts.GetKanisterToolsImage(),
		Command:      []string{"sh", "-c", "tail -f /dev/null"},
		PodOverride:  podOverride,
	}
	pr := kube.NewPodRunner(cli, options)
	podFunc := CheckRepositoryPodFunc(cli, tp, encryptionKey, targetPaths, insecureTLS)
	return pr.Run(ctx, podFunc)
}

func CheckRepositoryPodFunc(
	cli kubernetes.Interface,
	tp param.TemplateParams,
	encryptionKey,
	targetPath string,
	insecureTLS bool,
) func(ctx context.Context, pc kube.PodController) (map[string]interface{}, error) {
	return func(ctx context.Context, pc kube.PodController) (map[string]interface{}, error) {
		pod := pc.Pod()

		// Wait for pod to reach running state
		if err := pc.WaitForPodReady(ctx); err != nil {
			return nil, errors.Wrapf(err, "Failed while waiting for Pod %s to be ready", pod.Name)
		}

		remover, err := MaybeWriteProfileCredentials(ctx, pc, tp.Profile)
		if err != nil {
			return nil, err
		}

		// Parent context could already be dead, so removing file within new context
		defer remover.Remove(context.Background()) //nolint:errcheck

		err = restic.CheckIfRepoIsReachable(
			tp.Profile,
			targetPath,
			encryptionKey,
			insecureTLS,
			cli,
			pod.Namespace,
			pod.Name,
			pod.Spec.Containers[0].Name,
		)
		switch {
		case err == nil:
			break
		case strings.Contains(err.Error(), restic.PasswordIncorrect):
			return map[string]interface{}{
				CheckRepositoryPasswordIncorrect: "true",
				CheckRepositoryRepoDoesNotExist:  "false",
				FunctionOutputVersion:            kanister.DefaultVersion,
			}, nil
		case strings.Contains(err.Error(), restic.RepoDoesNotExist):
			return map[string]interface{}{
				CheckRepositoryPasswordIncorrect: "false",
				CheckRepositoryRepoDoesNotExist:  "true",
				FunctionOutputVersion:            kanister.DefaultVersion,
			}, nil
		default:
			return nil, err
		}
		return map[string]interface{}{
			CheckRepositoryPasswordIncorrect: "false",
			CheckRepositoryRepoDoesNotExist:  "false",
			FunctionOutputVersion:            kanister.DefaultVersion,
		}, nil
	}
}

func (c *CheckRepositoryFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	// Set progress percent
	c.progressPercent = progress.StartedPercent
	defer func() { c.progressPercent = progress.CompletedPercent }()

	var checkRepositoryArtifactPrefix, encryptionKey string
	var insecureTLS bool
	if err := Arg(args, CheckRepositoryArtifactPrefixArg, &checkRepositoryArtifactPrefix); err != nil {
		return nil, err
	}
	if err := OptArg(args, CheckRepositoryEncryptionKeyArg, &encryptionKey, restic.GeneratePassword()); err != nil {
		return nil, err
	}
	if err := OptArg(args, InsecureTLS, &insecureTLS, false); err != nil {
		return nil, err
	}

	podOverride, err := GetPodSpecOverride(tp, args, CheckRepositoryPodOverrideArg)
	if err != nil {
		return nil, err
	}

	if err = ValidateProfile(tp.Profile); err != nil {
		return nil, err
	}

	checkRepositoryArtifactPrefix = ResolveArtifactPrefix(checkRepositoryArtifactPrefix, tp.Profile)

	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	return CheckRepository(ctx, cli, tp, encryptionKey, checkRepositoryArtifactPrefix, CheckRepositoryJobPrefix, insecureTLS, podOverride)
}

func (*CheckRepositoryFunc) RequiredArgs() []string {
	return []string{CheckRepositoryArtifactPrefixArg}
}

func (*CheckRepositoryFunc) Arguments() []string {
	return []string{
		CheckRepositoryArtifactPrefixArg,
		CheckRepositoryEncryptionKeyArg,
		InsecureTLS,
	}
}
func (c *CheckRepositoryFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    c.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}
