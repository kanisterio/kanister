// Copyright 2024 The Kanister Authors.
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

package repository

import (
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/kanisterio/safecli/command"
	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal"
	"github.com/kanisterio/kanister/pkg/kopia/cli/repository/storage/azure"
	"github.com/kanisterio/kanister/pkg/kopia/cli/repository/storage/fs"
	"github.com/kanisterio/kanister/pkg/kopia/cli/repository/storage/gcs"
	"github.com/kanisterio/kanister/pkg/kopia/cli/repository/storage/s3"
	"github.com/kanisterio/kanister/pkg/log"
	rs "github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
)

var (
	cmdRepository = command.NewArgument("repository")

	subcmdCreate  = command.NewArgument("create")
	subcmdConnect = command.NewArgument("connect")
)

// optHostname creates a new option for the hostname of the repository.
func optHostname(h string) command.Applier {
	return command.NewOptionWithArgument("--override-hostname", h)
}

// optUsername creates a new option for the username of the repository.
func optUsername(u string) command.Applier {
	return command.NewOptionWithArgument("--override-username", u)
}

// optBlobRetention creates new blob retention options with a given mode and period.
// If mode is empty, the retention is disabled.
func optBlobRetention(mode string, period time.Duration) command.Applier {
	if mode == "" {
		return command.NewNoopArgument()
	}
	return command.NewArguments(
		command.NewOptionWithArgument("--retention-mode", mode),
		command.NewOptionWithArgument("--retention-period", period.String()),
	)
}

type storageBuilder func(internal.Location, string, log.Logger) command.Applier

var storageBuilders = map[rs.LocType]storageBuilder{
	rs.LocTypeFilestore:   fs.New,
	rs.LocTypeAzure:       azure.New,
	rs.LocTypeS3:          s3.New,
	rs.LocTypes3Compliant: s3.New,
	rs.LocTypeGCS:         gcs.New,
}

// optStorage creates a list of options for the specified storage location.
func optStorage(l internal.Location, repoPathPrefix string, logger log.Logger) command.Applier {
	sb := storageBuilders[l.Type()]
	if sb == nil {
		return errUnsupportedStorageType(l.Type())
	}
	return sb(l, repoPathPrefix, logger)
}

func errUnsupportedStorageType(t rs.LocType) command.Applier {
	err := errors.Wrapf(cli.ErrUnsupportedStorage, "storage location: %v", t)
	return command.NewErrorArgument(err)
}

// optReadOnly creates a new option for the read-only mode of the repository.
func optReadOnly(readOnly bool) command.Applier {
	return command.NewOption("--readonly", readOnly)
}

// optPointInTime creates a new option for the point-in-time of the repository.
func optPointInTime(tm strfmt.DateTime) command.Applier {
	if time.Time(tm).IsZero() {
		return command.NewNoopArgument()
	}
	return command.NewOptionWithArgument("--point-in-time", tm.String())
}
