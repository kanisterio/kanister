package server

import (
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag"
)

// TLSGenerateCert creates a new TLS generate certificate flag.
// if enable is false, the flag is not set.
func TLSGenerateCert(enable bool) flag.Applier {
	return flag.NewBoolFlag("--tls-generate-cert", enable)
}

// ServerAddress creates a new server address flag with a given address.
func ServerAddress(address string) flag.Applier {
	return flag.NewStringFlag("--address", address)
}

// ServerUsername creates a new server username flag with a given username.
func ServerUsername(username string) flag.Applier {
	return flag.NewStringFlag("--server-username", username)
}

// ServerControlUsername creates a new server control username flag with a given username.
func ServerControlUsername(username string) flag.Applier {
	return flag.NewStringFlag("--server-control-username", username)
}

// ServerPassword creates a new server password flag with a given password.
func ServerPassword(password string) flag.Applier {
	return flag.NewRedactedStringFlag("--server-password", password)
}

// ServerControlPassword creates a new server control password flag with a given password.
func ServerControlPassword(password string) flag.Applier {
	return flag.NewRedactedStringFlag("--server-control-password", password)
}

// TLSCertFile creates a new TLS certificate file flag with a given path.
func TLSCertFile(path string) flag.Applier {
	return flag.NewStringFlag("--tls-cert-file", path)
}

// TLSKeyFile creates a new TLS key file flag with a given path.
func TLSKeyFile(path string) flag.Applier {
	return flag.NewStringFlag("--tls-key-file", path)
}

const (
	shellRedirectToDevNull = "> /dev/null 2>&1"
	shellRunInBackground   = "&"
)

// Background flag enables running the server in the background.
func Background(enable bool) flag.Applier {
	if !enable {
		return flag.EmptyFlag()
	}
	return flag.NewFlags(
		flag.NewStringArgument(shellRedirectToDevNull),
		flag.NewStringArgument(shellRunInBackground),
	)
}

// ServerCertFingerprint creates a new server certificate fingerprint flag with a given fingerprint.
func ServerCertFingerprint(fingerprint string) flag.Applier {
	return flag.NewRedactedStringFlag("--server-cert-fingerprint", fingerprint)
}

// Username creates a new username argument with a given username.
func Username(username string) flag.Applier {
	return flag.NewStringArgument(username)
}

// UserPassword creates a new user password flag with a given password.
func UserPassword(password string) flag.Applier {
	return flag.NewRedactedStringFlag("--user-password", password)
}
