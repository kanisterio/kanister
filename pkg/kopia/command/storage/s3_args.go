package storage

import (
	"context"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/kanisterio/kanister/pkg/aws"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/logsafe"
	"github.com/kanisterio/kanister/pkg/secrets"
	"github.com/pkg/errors"
)

const (
	s3SubCommand           = "s3"
	s3AccessKeyFlag        = "--access-key"
	s3DisableTLSFlag       = "--disable-tls"
	s3DisableTLSVerifyFlag = "--disable-tls-verification"
	s3SecretAccessKeyFlag  = "--secret-access-key"
	s3SessionTokenFlag     = "--session-token"
	s3BucketFlag           = "--bucket"
	s3EndpointFlag         = "--endpoint"
	s3PrefixFlag           = "--prefix"
	s3RegionFlag           = "--region"
)

func kopiaS3Args(location, credentials map[string]string, assumeRoleDuration time.Duration, artifactPrefix string) (logsafe.Cmd, error) {
	args := logsafe.NewLoggable(s3SubCommand)
	args = args.AppendLoggableKV(s3BucketFlag, bucketName(location))

	e := endpoint(location)
	if e != "" {
		s3Endpoint := ResolveS3Endpoint(e)
		args = args.AppendLoggableKV(s3EndpointFlag, s3Endpoint)

		if HttpInsecureEndpoint(e) {
			args = args.AppendLoggable(s3DisableTLSFlag)
		}
	}

	artifactPrefix = GenerateFullRepoPath(prefix(location), artifactPrefix)

	credArgs, err := kopiaS3CredentialArgs(credentials, assumeRoleDuration)
	if err != nil {
		return nil, err
	}

	args = args.Combine(credArgs)
	args = args.AppendLoggableKV(s3PrefixFlag, artifactPrefix)

	if skipSSLVerify(location) {
		args = args.AppendLoggable(s3DisableTLSVerifyFlag)
	}

	region := region(location)
	if region != "" {
		args = args.AppendLoggableKV(s3RegionFlag, region)
	}

	return args, nil
}

func kopiaS3CredentialArgs(credentials map[string]string, assumeRoleDuration time.Duration) (logsafe.Cmd, error) {
	s3Creds, err := extractAWSCredentials(context.TODO(), credentials, assumeRoleDuration)
	if err != nil {
		return nil, err
	}

	args := logsafe.Cmd{}
	args = args.AppendRedactedKV(s3AccessKeyFlag, s3Creds.AccessKeyID)
	args = args.AppendRedactedKV(s3SecretAccessKeyFlag, s3Creds.SecretAccessKey)

	if s3Creds.SessionToken != "" {
		args = args.AppendRedactedKV(s3SessionTokenFlag, s3Creds.SessionToken)
	}

	return args, nil
}

func ResolveS3Endpoint(endpoint string) string {
	if strings.HasSuffix(endpoint, "/") {
		log.Debug().Print("Removing trailing slashes from the endpoint")
		endpoint = strings.TrimRight(endpoint, "/")
	}
	sp := strings.SplitN(endpoint, "://", 2)
	if len(sp) > 1 {
		log.Debug().Print("Removing leading protocol from the endpoint")
	}
	return sp[len(sp)-1]
}

func HttpInsecureEndpoint(endpoint string) bool {
	return strings.HasPrefix(endpoint, "http:")
}

func extractAWSCredentials(ctx context.Context, credsMap map[string]string, assumeRoleDuration time.Duration) (*credentials.Value, error) {
	if err := validateAWSCredentials(credsMap); err != nil {
		return nil, err
	}
	config := map[string]string{
		aws.AccessKeyID:        string(credsMap[secrets.AWSAccessKeyID]),
		aws.SecretAccessKey:    string(credsMap[secrets.AWSSecretAccessKey]),
		aws.ConfigRole:         string(credsMap[secrets.ConfigRole]),
		aws.AssumeRoleDuration: assumeRoleDuration.String(),
	}
	creds, err := aws.GetCredentials(ctx, config)
	if err != nil {
		return nil, err
	}
	val, err := creds.Get()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get AWS credentials")
	}
	exp, err := creds.ExpiresAt()
	if err == nil {
		log.Debug().Print("Credential expiration", field.M{"expirationTime": exp})
	}
	return &val, nil
}

func validateAWSCredentials(creds map[string]string) error {
	count := 0
	if _, ok := creds[secrets.AWSAccessKeyID]; ok {
		count++
	}
	if _, ok := creds[secrets.AWSSecretAccessKey]; ok {
		count++
	}
	if _, ok := creds[secrets.ConfigRole]; ok {
		count++
	}
	if len(creds) > count {
		return errors.New("Secret has an unknown field")
	}
	return nil
}
