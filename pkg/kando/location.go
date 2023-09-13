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

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/kanisterio/kanister/pkg/datamover"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	pathFlagName                         = "path"
	profileFlagName                      = "profile"
	repositoryServerFlagName             = "repository-server"
	repositoryServerUserHostnameFlagName = "repository-server-user-hostname"

	// DataMoverTypeProfile is used to specify that the DataMover is of type Profile
	DataMoverTypeProfile DataMoverType = "profile"
	// DataMoverTypeRepositoryServer is used to specify that the DataMover is of type RepositoryServer
	DataMoverTypeRepositoryServer DataMoverType = "repository-server"
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
	cmd.PersistentFlags().StringP(repositoryServerFlagName, "r", "", "Pass a Repository Server CR as a JSON string (required for kopia based blueprints)")
	cmd.PersistentFlags().StringP(repositoryServerUserHostnameFlagName, "c", "", "Pass the Repository Server Client Hostname (applicable if --repository-server is passed)")
	return cmd
}

func pathFlag(cmd *cobra.Command) string {
	return cmd.Flag(pathFlagName).Value.String()
}

// validateCommandArgs makes sure that we are getting exactly
// one of --profile or --repository-server flags
func validateCommandArgs(cmd *cobra.Command) error {
	profileFlag := cmd.Flags().Lookup(profileFlagName).Value.String()
	repositoryServerFlag := cmd.Flags().Lookup(repositoryServerFlagName).Value.String()
	if profileFlag != "" && repositoryServerFlag != "" {
		return errors.New("Either --profile or --repository-server should be provided")
	}
	if profileFlag == "" && repositoryServerFlag == "" {
		return errors.New("Please provide either --profile or --repository-server as per the datamover you want to use")
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
	case DataMoverTypeRepositoryServer:
		repositoryServerRef, err := unmarshalRepositoryServerFlag(cmd)
		if err != nil {
			return nil, err
		}
		return datamover.NewRepositoryServerDataMover(repositoryServerRef, outputName, kopiaSnapshot, cmd.Flag(repositoryServerUserHostnameFlagName).Value.String()), nil
	default:
		return nil, errors.New("Could not initialize DataMover.")
	}
}

func unmarshalProfileFlag(cmd *cobra.Command) (*param.Profile, error) {
	profileJSON := cmd.Flag(profileFlagName).Value.String()
	p := &param.Profile{}
	err := json.Unmarshal([]byte(profileJSON), p)
	return p, errors.Wrap(err, "failed to unmarshal profile")
}

func unmarshalRepositoryServerFlag(cmd *cobra.Command) (*param.RepositoryServer, error) {
	repositoryServerJSON := cmd.Flag(repositoryServerFlagName).Value.String()
	rs := &param.RepositoryServer{}
	err := json.Unmarshal([]byte(repositoryServerJSON), rs)
	return rs, errors.Wrap(err, "failed to unmarshal kopia repository server CR")
}

func dataMoverTypeFromCMD(c *cobra.Command) DataMoverType {
	profile := c.Flags().Lookup(profileFlagName).Value.String()
	if profile != "" {
		return DataMoverTypeProfile
	}
	repositoryServer := c.Flags().Lookup(repositoryServerFlagName).Value.String()
	if repositoryServer != "" {
		return DataMoverTypeRepositoryServer
	}
	return ""
}
