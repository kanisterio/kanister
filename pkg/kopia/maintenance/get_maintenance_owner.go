// Copyright 2022 The Kanister Authors.
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

package maintenance

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/kanisterio/errkit"
	kopiacli "github.com/kopia/kopia/cli"
	"github.com/kopia/kopia/repo/manifest"

	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kopia/command"
	"github.com/kanisterio/kanister/pkg/kube"
)

// KopiaUserProfile is a duplicate of struct for Kopia user profiles since Profile struct is in internal/user package and could not be imported
type KopiaUserProfile struct {
	ManifestID manifest.ID `json:"-"`

	Username            string `json:"username"`
	PasswordHashVersion int    `json:"passwordHashVersion"`
	PasswordHash        []byte `json:"passwordHash"`
}

// GetMaintenanceOwnerForConnectedRepository executes maintenance info command,
// and returns maintenance owner.
func GetMaintenanceOwnerForConnectedRepository(
	ctx context.Context,
	podController kube.PodController,
	configFilePath,
	logDirectory string,
) (string, error) {
	pod := podController.Pod()
	container := pod.Spec.Containers[0].Name
	commandExecutor, err := podController.GetCommandExecutor()
	if err != nil {
		return "", err
	}

	args := command.MaintenanceInfoCommandArgs{
		CommandArgs: &command.CommandArgs{
			ConfigFilePath: configFilePath,
			LogDirectory:   logDirectory,
		},
		GetJSONOutput: true,
	}
	cmd := command.MaintenanceInfo(args)

	var stdout, stderr bytes.Buffer
	err = commandExecutor.Exec(ctx, cmd, nil, &stdout, &stderr)
	format.LogWithCtx(ctx, pod.Name, container, stdout.String())
	format.LogWithCtx(ctx, pod.Name, container, stderr.String())
	if err != nil {
		return "", err
	}

	return parseOwner(stdout.Bytes())
}

func parseOwner(output []byte) (string, error) {
	maintInfo := kopiacli.MaintenanceInfo{}
	if err := json.Unmarshal(output, &maintInfo); err != nil {
		return "", errkit.Wrap(err, "failed to unmarshal maintenance info output")
	}
	return maintInfo.Owner, nil
}
