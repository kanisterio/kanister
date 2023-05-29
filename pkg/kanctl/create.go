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

package kanctl

import (
	"github.com/spf13/cobra"
)

const (
	dryRunFlag         = "dry-run"
	skipValidationFlag = "skip-validation"
)

func newCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a custom kanister resource",
	}
	cmd.AddCommand(newActionSetCmd())
	cmd.AddCommand(newProfileCommand())
	cmd.AddCommand(newRepositoryServerCommand())
	cmd.PersistentFlags().Bool(dryRunFlag, false, "if set, resource YAML will be printed but not created")
	cmd.PersistentFlags().Bool(skipValidationFlag, false, "if set, resource is not validated before creation")
	return cmd
}
