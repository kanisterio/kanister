package kopia

import (
	"context"
	"os"
	"time"

	"github.com/jpillora/backoff"
	"github.com/kopia/kopia/repo"
	"github.com/kopia/kopia/repo/content"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/poll"
)

const (
	defaultConnectMaxListCacheDuration time.Duration = time.Second * 600
	defaultConnectPersistCredentials                 = true
)

// ConnectToAPIServer connects to the Kopia API server running at the given address
func ConnectToAPIServer(
	ctx context.Context,
	cli kubernetes.Interface,
	tlsCertSecret,
	userPassphraseSecret *corev1.Secret,
	hostname,
	serverAddress,
	username string,
) error {
	// Extra fingerprint from the TLS Certificate secret
	fingerprint, err := ExtractFingerprintFromCertSecret(ctx, cli, tlsCertSecret.GetName(), tlsCertSecret.GetNamespace())
	if err != nil {
		return errors.Wrap(err, "Failed to extract fingerprint from Kopia API Server Certificate Secret")
	}

	// Extract user passphrase from the secret
	passphrase, ok := userPassphraseSecret.Data[hostname]
	if !ok {
		return errors.Errorf("Failed to extract client passphrase from secret. Secret: %s", userPassphraseSecret.GetName())
	}

	serverInfo := &repo.APIServerInfo{
		BaseURL:                             serverAddress,
		TrustedServerCertificateFingerprint: fingerprint,
	}

	opts := &repo.ConnectOptions{
		PersistCredentials: defaultConnectPersistCredentials,
		CachingOptions: content.CachingOptions{
			CacheDirectory:            defaultCacheDirectory,
			MaxCacheSizeBytes:         int64(defaultDataStoreGeneralContentCacheSizeMB << 20),
			MaxMetadataCacheSizeBytes: int64(defaultDataStoreGeneralMetadataCacheSizeMB << 20),
			MaxListCacheDurationSec:   int(defaultConnectMaxListCacheDuration.Seconds()),
		},
		HostnameOverride: hostname,
		UsernameOverride: username,
	}

	err = poll.WaitWithBackoff(ctx, backoff.Backoff{
		Factor: 2,
		Jitter: false,
		Min:    100 * time.Millisecond,
		Max:    180 * time.Second,
	}, func(c context.Context) (bool, error) {
		if err := repo.ConnectAPIServer(ctx, defaultConfigFilePath, serverInfo, string(passphrase), opts); err != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return errors.Wrap(err, "Failed to backup data to Kopia API server")
	}
	return nil
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
