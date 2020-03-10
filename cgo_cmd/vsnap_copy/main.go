// Copyright 2020 The Kanister Authors.
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

package main

import (
	"os"
	// "github.com/sirupsen/logrus"
	// "github.com/vmware-tanzu/astrolabe/pkg/astrolabe"
	// "github.com/vmware-tanzu/astrolabe/pkg/ivd"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	// crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	// kvm "github.com/kanisterio/kanister/pkg/blockstorage/vmware"
	// "github.com/kanisterio/kanister/pkg/location"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/param"
)

func main() {
	Execute()
}

const (
	pathFlagName    = "path"
	profileFlagName = "profile"
	vSphereCreds    = "vcreds"
)

func Execute() {
	root := newRootCommand()
	if err := root.Execute(); err != nil {
		log.WithError(err).Print("vsnapcopy failed to execute")
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vsnapcopy",
		Short: "push, pull from object storage",
	}
	cmd.AddCommand(newSnapshotPushCommand())
	//cmd.AddCommand(newSnapshotPullCommand())
	cmd.PersistentFlags().StringP(pathFlagName, "s", "", "Specify a path suffix (optional)")
	cmd.PersistentFlags().StringP(profileFlagName, "p", "", "Pass a Profile as a JSON string (required)")
	cmd.PersistentFlags().StringP(vSphereCreds, "v", "", "Pass vSphereCredentials as a JSON string (required)")
	_ = cmd.MarkFlagRequired(profileFlagName)
	_ = cmd.MarkFlagRequired(vSphereCreds)
	return cmd
}

func pathFlag(cmd *cobra.Command) string {
	return cmd.Flag(pathFlagName).Value.String()
}

func unmarshalProfileFlag(cmd *cobra.Command) (*param.Profile, error) {
	profileJSON := cmd.Flag(profileFlagName).Value.String()
	p := &param.Profile{}
	err := json.Unmarshal([]byte(profileJSON), p)
	return p, errors.Wrap(err, "failed to unmarshal profile")
}

type VSphereCreds struct {
	VCHost      string `json:"vchost"`
	VCUser      string `json:"vcuser"`
	VCPass      string `json:"vcpass"`
	VCS3UrlBase string `json:"s3urlbase"`
}

func unmarshalVSphereCredentials(cmd *cobra.Command) (*VSphereCreds, error) {
	credJSON := cmd.Flag(vSphereCreds).Value.String()
	creds := &VSphereCreds{}
	err := json.Unmarshal([]byte(credJSON), creds)
	return creds, errors.Wrap(err, "failed to unmarshal vsphere credentials")
}
