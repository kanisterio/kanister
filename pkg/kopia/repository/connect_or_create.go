package repository

import (
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/kopia/command"
	kerrors "github.com/kanisterio/kanister/pkg/kopia/errors"
)

// ConnectToOrCreateKopiaRepository connects to a kopia repository if present or creates if not already present
func ConnectToOrCreateKopiaRepository(
	cli kubernetes.Interface,
	namespace,
	pod,
	container string,
	cmdArgs command.RepositoryCommandArgs,
) error {
	err := ConnectToKopiaRepository(
		cli,
		namespace,
		pod,
		container,
		cmdArgs,
	)
	switch {
	case err == nil:
		// If repository connect was successful, we're done!
		return nil
	case kerrors.IsInvalidPasswordError(err):
		// If connect failed due to invalid password, no need to attempt creation
		return err
	}

	// Create a new repository
	err = CreateKopiaRepository(
		cli,
		namespace,
		pod,
		container,
		cmdArgs,
	)

	if err == nil {
		// Successfully created repository, we're done!
		return nil
	}

	// Creation failed. Repository may already exist or may have been
	// created by some parallel operation. Attempt connecting again.
	connectErr := ConnectToKopiaRepository(
		cli,
		namespace,
		pod,
		container,
		cmdArgs,
	)

	// Connected successfully after all
	if connectErr == nil {
		return nil
	}

	err = kerrors.Append(err, connectErr)
	return err
}
