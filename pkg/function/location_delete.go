// Copyright 2019 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package function

import (
	"context"
	"strings"

	"github.com/pkg/errors"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/location"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	// LocationDeleteArtifactArg provides the path to the artifacts on the object store
	LocationDeleteArtifactArg = "artifact"
)

func init() {
	kanister.Register(&locationDeleteFunc{})
}

var _ kanister.Func = (*locationDeleteFunc)(nil)

type locationDeleteFunc struct{}

func (*locationDeleteFunc) Name() string {
	return "LocationDelete"
}

func (*locationDeleteFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var artifact string
	var err error
	if err = Arg(args, LocationDeleteArtifactArg, &artifact); err != nil {
		return nil, err
	}
	// Validate the Profile
	if err = validateProfile(tp.Profile); err != nil {
		return nil, errors.Wrapf(err, "Failed to validate Profile")
	}
	return nil, location.Delete(ctx, *tp.Profile, strings.TrimPrefix(artifact, tp.Profile.Location.Bucket))
}

func (*locationDeleteFunc) RequiredArgs() []string {
	return []string{LocationDeleteArtifactArg}
}
