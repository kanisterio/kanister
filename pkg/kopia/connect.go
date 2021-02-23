// Copyright 2021 The Kanister Authors.
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

package kopia

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/jpillora/backoff"
	"github.com/kopia/kopia/repo"
	"github.com/kopia/kopia/repo/content"
	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/poll"
)

const (
	defaultConnectMaxListCacheDuration time.Duration = time.Second * 600
	defaultConnectPersistCredentials                 = true

	kopiaGetRepoParametersError = "unable to get repository parameters"
)

// ConnectToAPIServer connects to the Kopia API server running at the given address
func ConnectToAPIServer(
	ctx context.Context,
	tlsCert,
	userPassphrase,
	hostname,
	serverAddress,
	username string,
) error {
	// Extra fingerprint from the TLS Certificate secret
	fingerprint, err := ExtractFingerprintFromCertificateJSON(tlsCert)
	if err != nil {
		return errors.Wrap(err, "Failed to extract fingerprint from Kopia API Server Certificate Secret")
	}

	serverInfo := &repo.APIServerInfo{
		BaseURL:                             serverAddress,
		TrustedServerCertificateFingerprint: fingerprint,
		// TODO(@pavan): Remove once GRPC support is added
		DisableGRPC: true,
	}

	opts := &repo.ConnectOptions{
		PersistCredentials: defaultConnectPersistCredentials,
		CachingOptions: content.CachingOptions{
			CacheDirectory:            defaultCacheDirectory,
			MaxCacheSizeBytes:         int64(defaultDataStoreGeneralContentCacheSizeMB << 20),
			MaxMetadataCacheSizeBytes: int64(defaultDataStoreGeneralMetadataCacheSizeMB << 20),
			MaxListCacheDurationSec:   int(defaultConnectMaxListCacheDuration.Seconds()),
		},
		ClientOptions: repo.ClientOptions{
			Hostname: hostname,
			Username: username,
		},
	}

	//err = repo.ConnectAPIServer(ctx, defaultConfigFilePath, serverInfo, userPassphrase, opts)
	err = poll.WaitWithBackoff(ctx, backoff.Backoff{
		Factor: 2,
		Jitter: false,
		Min:    100 * time.Millisecond,
		Max:    15 * time.Second,
	}, func(c context.Context) (bool, error) {
		// TODO(@pavan): Modify this to use custom config file path, if required
		err := repo.ConnectAPIServer(ctx, defaultConfigFilePath, serverInfo, string(userPassphrase), opts)
		switch {
		case isGetRepoParametersError(err):
			log.Debug().WithError(err).Print("Connecting to the Kopia API Server")
			return false, nil
		case err != nil:
			return false, err
		}
		return true, nil
	})
	return errors.Wrap(err, "Failed connecting to the Kopia API Server")
}

// OpenRepository opens the connected Kopia repository
func OpenRepository(ctx context.Context) (repo.Repository, error) {
	if _, err := os.Stat(defaultConfigFilePath); os.IsNotExist(err) {
		return nil, errors.New("Failed find Kopia configuration file")
	}

	password, ok := repo.GetPersistedPassword(ctx, defaultConfigFilePath)
	if !ok || password == "" {
		return nil, errors.New("Failed to retrieve Kopia client passphrase")
	}

	r, err := repo.Open(ctx, defaultConfigFilePath, password, &repo.Options{})
	if os.IsNotExist(err) {
		return nil, errors.New("Failed to find Kopia repository, not connected to any repository")
	}
	if err != nil {
		return nil, errors.Wrap(err, "Failed to open Kopia repository")
	}

	return r, nil
}

func isGetRepoParametersError(err error) bool {
	return err != nil && strings.Contains(err.Error(), kopiaGetRepoParametersError)
}
