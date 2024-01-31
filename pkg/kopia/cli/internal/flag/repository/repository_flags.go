package repository

import (
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag"
)

// Hostname creates a new host flag with a given hostname.
func Hostname(hostname string) flag.Applier {
	return flag.NewStringFlag("--override-hostname", hostname)
}

// Username creates a new username flag with a given username.
func Username(username string) flag.Applier {
	return flag.NewStringFlag("--override-username", username)
}

// BlobRetentionFlag creates a new blob retention flag with a given mode and period.
// If mode is empty, the flag will be a no-op.
func BlobRetention(mode string, period time.Duration) flag.Applier {
	if mode == "" {
		return flag.DoNothingFlag()
	}
	return flag.NewFlags(
		flag.NewStringFlag("--retention-mode", mode),
		flag.NewStringFlag("--retention-period", period.String()),
	)
}

// PIT creates a new point-in-time flag with a given point-in-time.
// If pit is zero, the flag will be a no-op.
func PIT(pit strfmt.DateTime) flag.Applier {
	dt := strfmt.DateTime(pit)
	if time.Time(dt).IsZero() {
		return flag.DoNothingFlag()
	}
	return flag.NewStringFlag("--point-in-time", dt.String())
}

// ServerURL creates a new server URL flag with a given server URL.
func ServerURL(serverURL string) flag.Applier {
	return flag.NewStringFlag("--url", serverURL)
}

// ServerCertFingerprint creates a new server certificate fingerprint flag with a given fingerprint.
func ServerCertFingerprint(fingerprint string) flag.Applier {
	return flag.NewRedactedStringFlag("--server-cert-fingerprint", fingerprint)
}
