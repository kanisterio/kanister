package function

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/field"
	kankopia "github.com/kanisterio/kanister/pkg/kopia"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	// BackupDataAllUsingKopiaServerFuncName gives the name of the function
	BackupDataAllUsingKopiaServerFuncName = "BackupDataAllUsingKopiaServer"
	BackupDataAllUsingKopiaServerOutput   = "BackupAllUsingKopiaServerInfo"
)

type backupDataAllUsingKopiaServerFunc struct{}

func init() {
	_ = kanister.Register(&backupDataAllUsingKopiaServerFunc{})
}

func (*backupDataAllUsingKopiaServerFunc) Name() string {
	return BackupDataAllUsingKopiaServerFuncName
}

func (*backupDataAllUsingKopiaServerFunc) RequiredArgs() []string {
	return []string{
		BackupDataAllNamespaceArg,
		BackupDataAllContainerArg,
		BackupDataAllIncludePathArg,
		BackupDataAllBackupArtifactPrefixArg,
	}
}

func (*backupDataAllUsingKopiaServerFunc) Arguments() []string {
	return []string{
		BackupDataAllNamespaceArg,
		BackupDataAllContainerArg,
		BackupDataAllIncludePathArg,
		BackupDataAllPodsArg,
	}
}

func (*backupDataAllUsingKopiaServerFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var namespace, pods, container, includePath string
	var err error
	if err = Arg(args, BackupDataAllNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, BackupDataAllContainerArg, &container); err != nil {
		return nil, err
	}
	if err = Arg(args, BackupDataAllIncludePathArg, &includePath); err != nil {
		return nil, err
	}
	if err = OptArg(args, BackupDataAllPodsArg, &pods, ""); err != nil {
		return nil, err
	}

	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	var ps []string
	if pods == "" {
		switch {
		case tp.Deployment != nil:
			ps = tp.Deployment.Pods
		case tp.StatefulSet != nil:
			ps = tp.StatefulSet.Pods
		default:
			return nil, errors.New("Failed to get pods")
		}
	} else {
		ps = strings.Fields(pods)
	}
	ctx = field.Context(ctx, consts.ContainerNameKey, container)
	userPassphrase, cert, err := userCredentialsAndServerTLS(&tp)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch User Credentials / Certificate Data from Template Params")
	}

	fingerprint, err := kankopia.ExtractFingerprintFromCertificateJSON(cert)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch Kopia API Server Certificate Secret Data from Certificate")
	}

	hostname, userAccessPassphrase, err := hostNameAndUserPassPhraseFromRepoServer(userPassphrase)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch Hostname/User Passphrase from Secret")
	}
	return backupDataAllUsingKopiaServer(cli, ctx, namespace, container, includePath, hostname, tp.RepositoryServer.Address, fingerprint, tp.RepositoryServer.Username, userAccessPassphrase, ps)
}

func backupDataAllUsingKopiaServer(
	cli kubernetes.Interface,
	ctx context.Context,
	namespace,
	container,
	includePath,
	hostname,
	serverAddress,
	fingerprint,
	username,
	userPassphrase string,
	ps []string,
) (map[string]interface{}, error) {
	errChan := make(chan error, len(ps))
	outChan := make(chan BackupInfo, len(ps))
	Output := make(map[string]BackupInfo)

	for _, pod := range ps {
		go func(pod string, container string) {
			ctx = field.Context(ctx, consts.PodNameKey, pod)
			backupOutputs, err := backupDataUsingKopiaServer(cli, container, hostname, includePath, namespace, pod, serverAddress, fingerprint, username, userPassphrase, nil)
			errChan <- errors.Wrapf(err, "Failed to backup data for pod %s", pod)
			outChan <- BackupInfo{PodName: pod, BackupID: backupOutputs.SnapshotID}
		}(pod, container)
	}
	errs := make([]string, 0, len(ps))
	for i := 0; i < len(ps); i++ {
		err := <-errChan
		output := <-outChan
		if err != nil {
			errs = append(errs, err.Error())
		} else {
			Output[output.PodName] = output
		}
	}
	if len(errs) != 0 {
		return nil, errors.New(strings.Join(errs, "\n"))
	}
	manifestData, err := json.Marshal(Output)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to encode JSON data")
	}
	return map[string]interface{}{
		BackupDataAllUsingKopiaServerOutput: string(manifestData),
		FunctionOutputVersion:               kanister.DefaultVersion,
	}, nil
}
