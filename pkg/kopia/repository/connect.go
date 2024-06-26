package repository

import (
	"context"
	"strings"

	"github.com/kanisterio/errkit"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kopia/command"
	kerrors "github.com/kanisterio/kanister/pkg/kopia/errors"
	"github.com/kanisterio/kanister/pkg/kube"
)

// ConnectToKopiaRepository connects to an already existing kopia repository
func ConnectToKopiaRepository(
	ctx context.Context,
	cli kubernetes.Interface,
	namespace,
	pod,
	container string,
	cmdArgs command.RepositoryCommandArgs,
) error {
	cmd, err := command.RepositoryConnectCommand(cmdArgs)
	if err != nil {
		return errkit.Wrap(err, "Failed to generate repository connect command")
	}

	stdout, stderr, err := kube.Exec(ctx, cli, namespace, pod, container, cmd, nil)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)
	if err == nil {
		return nil
	}

	switch {
	case strings.Contains(stderr, kerrors.ErrInvalidPasswordStr):
		err = errkit.Wrap(err, kerrors.ErrInvalidPasswordStr)
	case err != nil && strings.Contains(err.Error(), kerrors.ErrCodeOutOfMemoryStr):
		err = errkit.Wrap(err, kerrors.ErrOutOfMemoryStr)
	case strings.Contains(stderr, kerrors.ErrAccessDeniedStr):
		err = errkit.Wrap(err, kerrors.ErrAccessDeniedStr)
	case kerrors.RepoNotInitialized(stderr):
		err = errkit.Wrap(err, kerrors.ErrRepoNotFoundStr)
	case kerrors.BucketDoesNotExist(stderr):
		err = errkit.Wrap(err, kerrors.ErrBucketDoesNotExistStr)
	}
	return errkit.Wrap(err, "Failed to connect to the backup repository")
}
