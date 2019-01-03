package location

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

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

//Delete data from location specified by `profile` and `suffix`.
func Delete(ctx context.Context, profile param.Profile, suffix string) error {
	switch profile.Location.Type {
	case crv1alpha1.LocationTypeS3Compliant:
		recursiveCmd, err := checkIfS3Dir(ctx, profile, suffix)
		if err != nil {
			return err
		}
		bin := s3CompliantBin()
		args := s3CompliantDeleteArgs(profile, suffix, recursiveCmd)
		env := s3CompliantEnv(profile)
		return deleteExec(ctx, bin, args, env)
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
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		// We could introduce rate-limiting by calling io.CopyN() in a loop.
		w, err := io.Copy(output, rc)
		if err != nil {
			log.WithError(err).Error("Failed to write data from pipe")
		}
		log.Infof("Read %d bytes", w)
		// rc may be closed already. Swallow close errors.
		_ = rc.Close()
	}()
	wg.Wait()
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
		// wc may be closed already. Swallow close errors.
		_ = wc.Close()
	}()
	return errors.Wrap(cmd.Wait(), "Failed to write data to location in profile")
}

func deleteExec(ctx context.Context, bin string, args []string, env []string) error {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = env
	err := cmd.Run()
	if err != nil {
		return errors.Wrap(err, "Failed to delete the artifact")
	}
	log.Info("Successfully deleted the artifact")
	return nil
}

func checkIfS3DirExec(ctx context.Context, bin string, args []string, env []string) (string, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = env
	out, err := cmd.Output()
	if err != nil {
		return "", errors.Wrap(err, "Failed to list the artifacts")
	}

	if bytes.Contains(out, []byte(" PRE ")) {
		// The path is a location of a directory in the S3 bucket
		// So append "--recursive" to the rm command
		return "--recursive", nil
	}
	return "", nil
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

func s3CompliantDeleteArgs(profile param.Profile, suffix string, recursive string) []string {
	target := s3CompliantPath(profile, suffix)
	return awsS3RmArgs(profile, target, recursive)
}

const s3Prefix = "s3://"

func s3CompliantPath(profile param.Profile, suffix string) string {
	path := filepath.Join(
		profile.Location.Bucket,
		profile.Location.Prefix,
		suffix,
	)
	if strings.HasPrefix(profile.Location.Bucket, s3Prefix) {
		return path
	}
	return s3Prefix + path
}

func s3CompliantEnv(profile param.Profile) []string {
	return awsCredsEnv(profile.Credential)
}

func awsS3CpArgs(profile param.Profile, src string, dst string) (cmd []string) {
	cmd = s3CompliantFlags(profile)
	cmd = append(cmd, "s3", "cp", src, dst)
	return cmd
}

func awsS3RmArgs(profile param.Profile, target string, recursiveCmd string) (cmd []string) {
	cmd = s3CompliantFlags(profile)
	cmd = append(cmd, "s3", "rm", target)
	if recursiveCmd != "" {
		cmd = append(cmd, recursiveCmd)
	}
	return cmd
}

const (
	AWSAccessKeyID     = "AWS_ACCESS_KEY_ID"
	AWSSecretAccessKey = "AWS_SECRET_ACCESS_KEY"
)

func awsCredsEnv(cred param.Credential) []string {
	if cred.Type != param.CredentialTypeKeyPair {
		panic("Unsupported Credential type: " + cred.Type)
	}
	return []string{
		fmt.Sprintf("%s=%s", AWSAccessKeyID, cred.KeyPair.ID),
		fmt.Sprintf("%s=%s", AWSSecretAccessKey, cred.KeyPair.Secret),
	}
}

func checkIfS3Dir(ctx context.Context, profile param.Profile, suffix string) (string, error) {
	var cmd []string
	target := s3CompliantPath(profile, suffix)
	cmd = s3CompliantFlags(profile)
	cmd = append(cmd, "s3", "ls", target)
	bin := s3CompliantBin()
	env := s3CompliantEnv(profile)
	return checkIfS3DirExec(ctx, bin, cmd, env)
}

func s3CompliantFlags(profile param.Profile) (cmd []string) {
	if profile.Location.Endpoint != "" {
		cmd = append(cmd, "--endpoint", profile.Location.Endpoint)
	}
	if profile.SkipSSLVerify {
		cmd = append(cmd, "--no-verify-ssl")
	}
	return cmd
}
