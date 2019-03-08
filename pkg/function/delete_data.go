package function

import (
	"context"

	"github.com/pkg/errors"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/location"
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

func (*deleteDataFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var artifact, namespace string
	var err error
	if err = Arg(args, DeleteDataArtifactArg, &artifact); err != nil {
		return nil, err
	}
	if err = OptArg(args, DeleteDataNamespaceArg, &namespace, ""); err != nil {
		return nil, err
	}
	// Validate the Profile
	if err = validateProfile(tp.Profile); err != nil {
		return nil, errors.Wrapf(err, "Failed to validate Profile")
	}

	return nil, location.Delete(ctx, *tp.Profile, artifact)
}

func (*deleteDataFunc) RequiredArgs() []string {
	return []string{DeleteDataArtifactArg}
}
