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
	"context"
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kopia"
	"github.com/kanisterio/kanister/pkg/location"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	kopiaSnapshotFlagName = "kopia-snapshot"
)

func newLocationPullCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull <target>",
		Short: "Pull from s3-compliant object storage to a file or stdout",
		Args:  cobra.ExactArgs(1),
		// TODO: Example invocations
		RunE: func(c *cobra.Command, args []string) error {
			return runLocationPull(c, args)
		},
	}
	cmd.Flags().StringP(kopiaSnapshotFlagName, "k", "", "Pass the kopia snapshot information from the location push command (optional)")
	return cmd
}

func kopiaSnapshotFlag(cmd *cobra.Command) string {
	return cmd.Flag(kopiaSnapshotFlagName).Value.String()
}

func runLocationPull(cmd *cobra.Command, args []string) error {
	target, err := targetWriter(args[0])
	if err != nil {
		return err
	}
	p, err := unmarshalProfileFlag(cmd)
	if err != nil {
		return err
	}
	s := pathFlag(cmd)
	ctx := context.Background()
	if p.Location.Type == crv1alpha1.LocationTypeKopia {
		snapJSON := kopiaSnapshotFlag(cmd)
		if snapJSON == "" {
			return errors.New("kopia snapshot information is required to pull data using kopia")
		}
		kopiaSnap, err := kopia.UnmarshalKopiaSnapshot(snapJSON)
		if err != nil {
			return err
		}
		if err = connectToKopiaServer(ctx, p); err != nil {
			return err
		}
		return kopiaLocationPull(ctx, kopiaSnap.ID, s, target, p.Credential.KopiaServerSecret.Password)
	}
	return locationPull(ctx, p, s, target)
}

func targetWriter(target string) (io.Writer, error) {
	if target != usePipeParam {
		return os.OpenFile(target, os.O_RDWR|os.O_CREATE, 0755)
	}
	return os.Stdout, nil
}

func locationPull(ctx context.Context, p *param.Profile, path string, target io.Writer) error {
	return location.Read(ctx, target, *p, path)
}

// kopiaLocationPull pulls the data from a kopia snapshot into the given target
func kopiaLocationPull(ctx context.Context, backupID, path string, target io.Writer, password string) error {
	return kopia.Read(ctx, backupID, path, target, password)
}

// connectToKopiaServer connects to the kopia server with given creds
func connectToKopiaServer(ctx context.Context, kp *param.Profile) error {
	contentCacheSize := kopia.GetDataStoreGeneralContentCacheSize(kp.Credential.KopiaServerSecret.ConnectOptions)
	metadataCacheSize := kopia.GetDataStoreGeneralMetadataCacheSize(kp.Credential.KopiaServerSecret.ConnectOptions)
	return kopia.ConnectToAPIServer(
		ctx,
		kp.Credential.KopiaServerSecret.Cert,
		kp.Credential.KopiaServerSecret.Password,
		kp.Credential.KopiaServerSecret.Hostname,
		kp.Location.Endpoint,
		kp.Credential.KopiaServerSecret.Username,
		contentCacheSize,
		metadataCacheSize,
	)
}
