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

package kando

import (
	"encoding/json"

	"github.com/kanisterio/errkit"
	"github.com/spf13/cobra"

	"github.com/kanisterio/kanister/pkg/datamover"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	pathFlagName    = "path"
	profileFlagName = "profile"

	// DataMoverTypeProfile is used to specify that the DataMover is of type Profile
	DataMoverTypeProfile DataMoverType = "profile"
)

type DataMoverType string

func newLocationCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "location <command>",
		Short: "Push, pull and delete from object storage",
	}
	cmd.AddCommand(newLocationPushCommand())
	cmd.AddCommand(newLocationPullCommand())
	cmd.AddCommand(newLocationDeleteCommand())
	cmd.PersistentFlags().StringP(pathFlagName, "s", "", "Specify a path suffix (optional)")
	cmd.PersistentFlags().StringP(profileFlagName, "p", "", "Pass a Profile as a JSON string (required)")
	return cmd
}

func pathFlag(cmd *cobra.Command) string {
	return cmd.Flag(pathFlagName).Value.String()
}

// validateCommandArgs makes sure that the --profile flag is provided
func validateCommandArgs(cmd *cobra.Command) error {
	profileFlag := cmd.Flags().Lookup(profileFlagName).Value.String()
	if profileFlag == "" {
		return errkit.New("Please provide the --profile flag")
	}
	return nil
}

// dataMoverForKopiaSnapshotFlag returns a DataMover based on the --kopia-snapshot flag
func dataMoverForKopiaSnapshotFlag(cmd *cobra.Command) (datamover.DataMover, error) {
	return dataMoverFromCMD(cmd, cmd.Flag(kopiaSnapshotFlagName).Value.String(), "")
}

// dataMoverForOutputNameFlag returns a DataMover based on the --output-name flag
func dataMoverForOutputNameFlag(cmd *cobra.Command) (datamover.DataMover, error) {
	return dataMoverFromCMD(cmd, "", cmd.Flag(outputNameFlagName).Value.String())
}

func dataMoverFromCMD(cmd *cobra.Command, kopiaSnapshot, outputName string) (datamover.DataMover, error) {
	switch dataMoverTypeFromCMD(cmd) {
	case DataMoverTypeProfile:
		profileRef, err := unmarshalProfileFlag(cmd)
		if err != nil {
			return nil, err
		}
		return datamover.NewProfileDataMover(profileRef, outputName, kopiaSnapshot), nil
	default:
		return nil, errkit.New("Could not initialize DataMover.")
	}
}

func unmarshalProfileFlag(cmd *cobra.Command) (*param.Profile, error) {
	profileJSON := cmd.Flag(profileFlagName).Value.String()
	p := &param.Profile{}
	err := json.Unmarshal([]byte(profileJSON), p)
	return p, errkit.Wrap(err, "failed to unmarshal profile")
}

func dataMoverTypeFromCMD(c *cobra.Command) DataMoverType {
	profile := c.Flags().Lookup(profileFlagName).Value.String()
	if profile != "" {
		return DataMoverTypeProfile
	}
	return ""
}
