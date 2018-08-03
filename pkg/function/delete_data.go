package function

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	// DeleteDataNamespaceArg provides the namespace
	DeleteDataNamespaceArg = "namespace"
	// DeleteDataArtifactArg provides the path to the artifacts on the object store
	DeleteDataArtifactArg = "artifact"
)

func init() {
	kanister.Register(&deleteDataFunc{})
}

var _ kanister.Func = (*deleteDataFunc)(nil)

type deleteDataFunc struct{}

func (*deleteDataFunc) Name() string {
	return "DeleteData"
}

func generateDeleteCommand(artifact string, profile *param.Profile) []string {
	// Command to export credentials
	cmd := []string{"export", fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s\n", profile.Credential.KeyPair.Secret)}
	cmd = append(cmd, "export", fmt.Sprintf("AWS_ACCESS_KEY_ID=%s\n", profile.Credential.KeyPair.ID))
	// Command to delete from the object store
	cmd = append(cmd, "aws")
	if profile.Location.S3Compliant.Endpoint != "" {
		cmd = append(cmd, "--endpoint", profile.Location.S3Compliant.Endpoint)
	}
	if profile.SkipSSLVerify {
		cmd = append(cmd, "--no-verify-ssl")
	}
	cmd = append(cmd, "s3", "rm", artifact)
	command := strings.Join(cmd, " ")
	return []string{"bash", "-o", "errexit", "-o", "pipefail", "-c", command}
}

func (*deleteDataFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) error {
	var artifact, namespace string
	var err error
	if err = Arg(args, DeleteDataArtifactArg, &artifact); err != nil {
		return err
	}
	if err = OptArg(args, DeleteDataNamespaceArg, &namespace, ""); err != nil {
		return err
	}
	// Validate the Profile
	if err = validateProfile(tp.Profile); err != nil {
		return errors.Wrapf(err, "Failed to validate Profile")
	}
	// Generate delete command
	cmd := generateDeleteCommand(artifact, tp.Profile)
	// Use KubeTask to delete the artifact
	return kubeTask(ctx, namespace, "kanisterio/kanister-tools:0.10.0", cmd)
}

func (*deleteDataFunc) RequiredArgs() []string {
	return []string{DeleteDataArtifactArg}
}
