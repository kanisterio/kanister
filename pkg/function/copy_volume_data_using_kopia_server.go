// Copyright 2023 The Kanister Authors.
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

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	CopyVolumeDataUsingKopiaServerFuncName = "CopyVolumeDataUsingKopiaServer"
)

type copyVolumeDataUsingKopiaServerFunc struct{}

func init() {
	err := kanister.Register(&copyVolumeDataUsingKopiaServerFunc{})
	if err != nil {
		return
	}
}

var _ kanister.Func = (*copyVolumeDataUsingKopiaServerFunc)(nil)

func (*copyVolumeDataUsingKopiaServerFunc) Name() string {
	return CopyVolumeDataUsingKopiaServerFuncName
}

func (f *copyVolumeDataUsingKopiaServerFunc) RequiredArgs() []string {
	//TODO implement me
	panic("implement me")
}

func (f *copyVolumeDataUsingKopiaServerFunc) Arguments() []string {
	//TODO implement me
	panic("implement me")
}

func (f *copyVolumeDataUsingKopiaServerFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]any) (map[string]any, error) {
	//TODO implement me
	panic("implement me")
}
