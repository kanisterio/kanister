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

// Write pipes data from `in` into the location specified by `profile` and `suffix`.
func Write(ctx context.Context, in io.Reader, profile param.Profile, suffix string) error {
	switch profile.Location.Type {
	case crv1alpha1.LocationTypeS3Compliant:
		bin := s3CompliantBin()
		args := s3CompliantWriteArgs(profile, suffix)
		env := s3CompliantEnv(profile)
		return writeExec(ctx, in, bin, args, env)
	}
	return errors.Errorf("Unsupported Location type: %s", profile.Location.Type)
}

// Read pipes data from `in` into the location specified by `profile` and `suffix`.
func Read(ctx context.Context, out io.Writer, profile param.Profile, suffix string) error {
	switch profile.Location.Type {
	case crv1alpha1.LocationTypeS3Compliant:
		bin := s3CompliantBin()
		args := s3CompliantReadArgs(profile, suffix)
		env := s3CompliantEnv(profile)
		return readExec(ctx, out, bin, args, env)
	}
	return errors.Errorf("Unsupported Location type: %s", profile.Location.Type)
}

func readExec(ctx context.Context, output io.Writer, bin string, args []string, env []string) error {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = env
	rc, err := cmd.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "Failed to setup data pipe")
	}
	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "Failed to start read-data command")
	}
	go func() {
		// We could introduce rate-limiting by calling io.CopyN() in a loop.
		w, err := io.Copy(output, rc)
		if err != nil {
			log.WithError(err).Error("Failed to write data from pipe")
		}
		log.Infof("Read %d bytes", w)
		if err := rc.Close(); err != nil {
			log.WithError(err).Error("Failed to close pipe")
		}
	}()
	return errors.Wrap(cmd.Wait(), "Failed to read data from location in profile")
}

func writeExec(ctx context.Context, input io.Reader, bin string, args []string, env []string) error {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = env
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

func s3CompliantBin() string {
	return awsBin
}

func s3CompliantReadArgs(profile param.Profile, suffix string) []string {
	src := s3CompliantPath(profile, suffix)
	return awsS3CpArgs(profile, src, "-")
}

func s3CompliantWriteArgs(profile param.Profile, suffix string) []string {
	dst := s3CompliantPath(profile, suffix)
	return awsS3CpArgs(profile, "-", dst)
}

func s3CompliantPath(profile param.Profile, suffix string) string {
	return filepath.Join(
		profile.Location.S3Compliant.Bucket,
		profile.Location.S3Compliant.Prefix,
		suffix,
	)
}

func s3CompliantEnv(profile param.Profile) []string {
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
