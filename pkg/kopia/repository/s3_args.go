package repository

import (
	"context"
	"strings"
	"time"

	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/logsafe"
	"github.com/kanisterio/kanister/pkg/secrets"
	v1 "k8s.io/api/core/v1"
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

func kopiaS3Args(locationSecret, locationCredSecret *v1.Secret, assumeRoleDuration time.Duration, artifactPrefix string) (logsafe.Cmd, error) {
	args := logsafe.NewLoggable(s3SubCommand)
	args = args.AppendLoggableKV(s3BucketFlag, bucketName(locationSecret))

	e := endpoint(locationSecret)
	if e != "" {
		s3Endpoint := ResolveS3Endpoint(e)
		args = args.AppendLoggableKV(s3EndpointFlag, s3Endpoint)

		if HttpInsecureEndpoint(e) {
			args = args.AppendLoggable(s3DisableTLSFlag)
		}
	}

	artifactPrefix = GenerateFullRepoPath(prefix(locationSecret), artifactPrefix)

	credArgs, err := kopiaS3CredentialArgs(locationCredSecret, assumeRoleDuration)
	if err != nil {
		return nil, err
	}

	args = args.Combine(credArgs)
	args = args.AppendLoggableKV(s3PrefixFlag, artifactPrefix)

	if skipSSLVerify(locationSecret) {
		args = args.AppendLoggable(s3DisableTLSVerifyFlag)
	}

	region := region(locationSecret)
	if region != "" {
		args = args.AppendLoggableKV(s3RegionFlag, region)
	}

	return args, nil
}

func kopiaS3CredentialArgs(locationCredSecret *v1.Secret, assumeRoleDuration time.Duration) (logsafe.Cmd, error) {
	s3Creds, err := secrets.ExtractAWSCredentials(context.TODO(), locationCredSecret, assumeRoleDuration)
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
