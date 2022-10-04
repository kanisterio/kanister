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
	"strings"

	"github.com/kopia/kopia/repo/manifest"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

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

// GetMaintenanceOwnerForConnectedRepository executes maintenance info command, parses output
// and returns maintenance owner
func GetMaintenanceOwnerForConnectedRepository(
	cli kubernetes.Interface,
	namespace,
	pod,
	container,
	repoPassword,
	configFilePath,
	logDirectory string,
) (string, error) {
	args := command.MaintenanceInfoCommandArgs{
		CommandArgs: &command.CommandArgs{
			RepoPassword:   repoPassword,
			ConfigFilePath: configFilePath,
			LogDirectory:   logDirectory,
		},
		GetJsonOutput: false,
	}
	cmd := command.MaintenanceInfo(args)
	stdout, stderr, err := kube.Exec(cli, namespace, pod, container, cmd, nil)
	format.Log(pod, container, stdout)
	format.Log(pod, container, stderr)
	if err != nil {
		return "", err
	}
	parsedOwner := parseOutput(stdout)
	if parsedOwner == "" {
		return "", errors.New("Failed parsing maintenance info output to get owner")
	}
	return parsedOwner, nil
}

func parseOutput(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Owner") {
			arr := strings.Split(line, ":")
			if len(arr) == 2 {
				return strings.TrimSpace(arr[1])
			}
		}
	}
	return ""
}
