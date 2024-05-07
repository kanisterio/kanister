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
	"github.com/kanisterio/safecli"
	"github.com/kanisterio/safecli/command"

	"github.com/kanisterio/kanister/pkg/kopia/cli/args"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/opts"
	"github.com/kanisterio/kanister/pkg/log"
)

// ConnectServerArgs defines the arguments for the `kopia repository connect server` command.
type ConnectServerArgs struct {
	args.Common
	args.Cache

	Hostname    string // hostname of the repository
	Username    string // username of the repository
	ServerURL   string // URL of the Kopia Repository API server
	Fingerprint string // fingerprint of the server's TLS certificate
	ReadOnly    bool   // connect to a repository in read-only mode

	Logger log.Logger
}

// ConnectServer creates a new `kopia repository connect server...` command.
func ConnectServer(connectServerArgs ConnectServerArgs) (*safecli.Builder, error) {
	appliers := []command.Applier{
		opts.Common(connectServerArgs.Common),
		cmdRepository, subcmdConnect, subcmdServer,
		opts.CheckForUpdates(false),
		optReadOnly(connectServerArgs.ReadOnly),
		opts.Cache(connectServerArgs.Cache),
		optHostname(connectServerArgs.Hostname),
		optUsername(connectServerArgs.Username),
		optServerURL(connectServerArgs.ServerURL),
		optServerCertFingerprint(connectServerArgs.Fingerprint),
	}
	appliers = append(appliers, args.RepositoryConnectServer.CommandAppliers()...)
	return internal.NewKopiaCommand(
		appliers...,
	)
}
