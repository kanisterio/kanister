package repository

import (
	"strings"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kopia/command"
	kerrors "github.com/kanisterio/kanister/pkg/kopia/errors"
	"github.com/kanisterio/kanister/pkg/kube"
)

// ConnectToKopiaRepository connects to an already existing kopia repository
func ConnectToKopiaRepository(
	cli kubernetes.Interface,
	namespace,
	pod,
	container string,
	cmdArgs command.RepositoryCommandArgs,
) error {
	cmd, err := command.RepositoryConnectCommand(cmdArgs)
	if err != nil {
		return errors.Wrap(err, "Failed to generate repository connect command")
	}

	stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd, nil)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)
	if err == nil {
		return nil
	}

	switch {
	case strings.Contains(stderr, kerrors.ErrInvalidPasswordStr):
		err = errors.WithMessage(err, kerrors.ErrInvalidPasswordStr)
	case err != nil && strings.Contains(err.Error(), kerrors.ErrCodeOutOfMemoryStr):
		err = errors.WithMessage(err, kerrors.ErrOutOfMemoryStr)
	case strings.Contains(stderr, kerrors.ErrAccessDeniedStr):
		err = errors.WithMessage(err, kerrors.ErrAccessDeniedStr)
	case kerrors.RepoNotInitialized(stderr):
		err = errors.WithMessage(err, kerrors.ErrRepoNotFoundStr)
	case kerrors.BucketDoesNotExist(stderr):
		err = errors.WithMessage(err, kerrors.ErrBucketDoesNotExistStr)
	}
	return errors.Wrap(err, "Failed to connect to the backup repository")
}
