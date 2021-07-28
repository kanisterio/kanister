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
	"github.com/kanisterio/kanister/pkg/output"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	outputNameFlagName    = "output-name"
	defaultKandoOutputKey = "kandoOutput"
)

func newLocationPushCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push <source>",
		Short: "Push a source file or stdin stream to s3-compliant object storage",
		Args:  cobra.ExactArgs(1),
		// TODO: Example invocations
		RunE: func(c *cobra.Command, args []string) error {
			return runLocationPush(c, args)
		},
	}
	cmd.Flags().StringP(outputNameFlagName, "o", defaultKandoOutputKey, "Specify a name to be used for the output produced by kando. Set to `kandoOutput` by default")
	return cmd
}

func outputNameFlag(cmd *cobra.Command) string {
	return cmd.Flag(outputNameFlagName).Value.String()
}

func runLocationPush(cmd *cobra.Command, args []string) error {
	p, err := unmarshalProfileFlag(cmd)
	if err != nil {
		return err
	}
	s := pathFlag(cmd)
	ctx := context.Background()
	if p.Location.Type == crv1alpha1.LocationTypeKopia {
		outputName := outputNameFlag(cmd)
		if err = connectToKopiaServer(ctx, p); err != nil {
			return err
		}
		return kopiaLocationPush(ctx, s, outputName, args[0], p.Credential.KopiaServerSecret.Password)
	}
	source, err := sourceReader(args[0])
	if err != nil {
		return err
	}
	return locationPush(ctx, p, s, source)
}

const usePipeParam = `-`

func sourceReader(source string) (io.Reader, error) {
	if source != usePipeParam {
		return os.Open(source)
	}
	fi, err := os.Stdin.Stat()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to Stat stdin")
	}
	if fi.Mode()&os.ModeNamedPipe == 0 {
		return nil, errors.New("Stdin must be piped when the source parameter is \"-\"")
	}
	return os.Stdin, nil
}

func locationPush(ctx context.Context, p *param.Profile, path string, source io.Reader) error {
	return location.Write(ctx, source, *p, path)
}

// kopiaLocationPush pushes the data from the source using a kopia snapshot
func kopiaLocationPush(ctx context.Context, path, outputName string, sourcePath, password string) error {
	var snapInfo *kopia.SnapshotInfo
	var err error
	switch sourcePath {
	case usePipeParam:
		snapInfo, err = kopia.Write(ctx, path, os.Stdin, password)
	default:
		snapInfo, err = kopia.WriteFile(ctx, path, sourcePath, password)
	}
	if err != nil {
		return errors.Wrap(err, "Failed to push data using kopia")
	}

	snapInfoJSON, err := kopia.MarshalKopiaSnapshot(snapInfo)
	if err != nil {
		return err
	}

	return output.PrintOutput(outputName, snapInfoJSON)
}
