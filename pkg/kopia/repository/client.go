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

package repository

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jpillora/backoff"
	"github.com/kopia/kopia/repo"
	"github.com/kopia/kopia/repo/content"
	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/kopia"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/poll"
)

const (
	defaultConnectMaxListCacheDuration time.Duration = time.Second * 600
	kopiaGetRepoParametersError                      = "unable to get repository parameters"
)

// ConnectToAPIServer connects to the Kopia API server running at the given address
func ConnectToAPIServer(
	ctx context.Context,
	tlsCert,
	userPassphrase,
	hostname,
	serverAddress,
	username string,
	contentCacheMB,
	metadataCacheMB int,
) error {
	// Extra fingerprint from the TLS Certificate secret
	fingerprint, err := kopia.ExtractFingerprintFromCertificate(tlsCert)
	if err != nil {
		return errors.Wrap(err, "Failed to extract fingerprint from Kopia API Server Certificate Secret")
	}

	serverInfo := &repo.APIServerInfo{
		BaseURL:                             serverAddress,
		TrustedServerCertificateFingerprint: fingerprint,
		// TODO(@pavan): Remove once GRPC support is added (kopia 0.8 release)
		DisableGRPC: true,
	}

	opts := &repo.ConnectOptions{
		CachingOptions: content.CachingOptions{
			CacheDirectory:              kopia.DefaultClientCacheDirectory,
			ContentCacheSizeLimitBytes:  int64(contentCacheMB << 20),
			ContentCacheSizeBytes:       int64(contentCacheMB << 20),
			MetadataCacheSizeBytes:      int64(metadataCacheMB << 20),
			MetadataCacheSizeLimitBytes: int64(metadataCacheMB << 20),
			MaxListCacheDuration:        content.DurationSeconds(defaultConnectMaxListCacheDuration.Seconds()),
		},
		ClientOptions: repo.ClientOptions{
			Hostname: hostname,
			Username: username,
		},
	}

	err = poll.WaitWithBackoff(ctx, backoff.Backoff{
		Factor: 2,
		Jitter: false,
		Min:    100 * time.Millisecond,
		Max:    15 * time.Second,
	}, func(c context.Context) (bool, error) {
		// TODO(@pavan): Modify this to use custom config file path, if required
		err := repo.ConnectAPIServer(ctx, kopia.DefaultClientConfigFilePath, serverInfo, userPassphrase, opts)
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

// Open connects to the kopia repository based on the config stored in the config file
// NOTE: This assumes that `kopia repository connect` has been already run on the machine
// OR the above Connect function has been used to connect to the repository server
func Open(ctx context.Context, configFile, password, purpose string) (repo.RepositoryWriter, error) {
	repoConfig := repositoryConfigFileName(configFile)
	if _, err := os.Stat(repoConfig); os.IsNotExist(err) {
		return nil, errors.New("Failed find kopia configuration file")
	}

	r, err := repo.Open(ctx, repoConfig, password, &repo.Options{})
	if os.IsNotExist(err) {
		return nil, errors.New("Failed to find kopia repository, use `kopia repository create` or kopia repository connect` if already created")
	}

	if err != nil {
		return nil, errors.Wrap(err, "Failed to open kopia repository")
	}

	_, rw, err := r.NewWriter(ctx, repo.WriteSessionOptions{
		Purpose:  purpose,
		OnUpload: func(i int64) {},
	})

	return rw, errors.Wrap(err, "Failed to open kopia repository writer")
}

func repositoryConfigFileName(configFile string) string {
	if configFile != "" {
		return configFile
	}
	return filepath.Join(os.Getenv("HOME"), ".config", "kopia", "repository.config")
}

func isGetRepoParametersError(err error) bool {
	return err != nil && strings.Contains(err.Error(), kopiaGetRepoParametersError)
}
