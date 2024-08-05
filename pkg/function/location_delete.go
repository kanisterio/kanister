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
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/location"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
)

const (
	// LocationDeleteFuncName gives the function name
	LocationDeleteFuncName = "LocationDelete"
	// LocationDeleteArtifactArg provides the path to the artifacts on the object store
	LocationDeleteArtifactArg = "artifact"
)

func init() {
	_ = kanister.Register(&locationDeleteFunc{})
}

var _ kanister.Func = (*locationDeleteFunc)(nil)

type locationDeleteFunc struct {
	progressPercent string
}

func (*locationDeleteFunc) Name() string {
	return LocationDeleteFuncName
}

func (l *locationDeleteFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	// Set progress percent
	l.progressPercent = progress.StartedPercent
	defer func() { l.progressPercent = progress.CompletedPercent }()

	var artifact string
	var err error
	if err = Arg(args, LocationDeleteArtifactArg, &artifact); err != nil {
		return nil, err
	}
	if err = ValidateProfile(tp.Profile); err != nil {
		return nil, errors.Wrapf(err, "Failed to validate Profile")
	}
	return nil, location.Delete(ctx, *tp.Profile, strings.TrimPrefix(artifact, tp.Profile.Location.Bucket))
}

func (*locationDeleteFunc) RequiredArgs() []string {
	return []string{LocationDeleteArtifactArg}
}

func (*locationDeleteFunc) Arguments() []string {
	return []string{LocationDeleteArtifactArg}
}

func (l *locationDeleteFunc) Validate(args map[string]any) error {
	if err := utils.CheckSupportedArgs(l.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(l.RequiredArgs(), args)
}

func (l *locationDeleteFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    l.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}
