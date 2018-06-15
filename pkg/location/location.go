package location

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/param"
)

// Write pipes data from `r` into the location specified by `profile` and `suffix`.
func Write(ctx context.Context, r io.Reader, profile param.Profile, suffix string) error {
	b := bin(profile)
	a := args(profile, suffix)
	e := env(profile)
	return write(ctx, r, b, a, e)
}

func write(ctx context.Context, input io.Reader, binary string, arguments []string, environment []string) error {
	cmd := exec.CommandContext(ctx, binary, arguments...)
	cmd.Env = environment
	wc, err := cmd.StdinPipe()
	if err != nil {
		return errors.Wrap(err, "Failed to setup data pipe")
	}
	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "Failed to start write-data command")
	}
	go func() {
		// We could introduce rate-limiting by calling io.CopyN() in a loop.
		w, err := io.Copy(wc, input)
		if err != nil {
			log.WithError(err).Error("Failed to write data from pipe")
		}
		log.Infof("Wrote %d bytes", w)
		if err := wc.Close(); err != nil {
			log.WithError(err).Error("Failed to close pipe")
		}
	}()
	return errors.Wrap(cmd.Wait(), "Failed to write data to location in profile")
}

const awsBin = `aws`

func bin(profile param.Profile) string {
	if profile.Location.Type != crv1alpha1.LocationTypeS3Compliant {
		panic("Unsupported Location type: " + profile.Location.Type)
	}
	return awsBin
}

func args(profile param.Profile, suffix string) []string {
	if profile.Location.Type != crv1alpha1.LocationTypeS3Compliant {
		panic("Unsupported Location type: " + profile.Location.Type)
	}
	dst := filepath.Join(
		profile.Location.S3Compliant.Bucket,
		profile.Location.S3Compliant.Prefix,
		suffix,
	)
	return awsS3CpArgs(profile, "-", dst)
}

func env(profile param.Profile) []string {
	if profile.Location.Type != crv1alpha1.LocationTypeS3Compliant {
		panic("Unsupported Location type: " + profile.Location.Type)
	}
	return awsCredsEnv(profile.Credential)
}

func awsS3CpArgs(profile param.Profile, src string, dst string) (cmd []string) {
	if profile.Location.S3Compliant.Endpoint != "" {
		cmd = append(cmd, "--endpoint", profile.Location.S3Compliant.Endpoint)
	}
	if profile.SkipSSLVerify {
		cmd = append(cmd, "--no-verify-ssl")
	}
	cmd = append(cmd, "s3", "cp", src, dst)
	return cmd
}

const (
	awsAccessKeyID     = "AWS_ACCESS_KEY_ID"
	awsSecretAccessKey = "AWS_SECRET_ACCESS_KEY"
)

func awsCredsEnv(cred param.Credential) []string {
	if cred.Type != param.CredentialTypeKeyPair {
		panic("Unsupported Credential type: " + cred.Type)
	}
	return []string{
		fmt.Sprintf("%s=%s", awsAccessKeyID, cred.KeyPair.ID),
		fmt.Sprintf("%s=%s", awsSecretAccessKey, cred.KeyPair.Secret),
	}
}
