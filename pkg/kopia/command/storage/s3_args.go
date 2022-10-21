package storage

import (
	"strings"
	"time"

	"github.com/kanisterio/kanister/pkg/log"
	"github.com/kanisterio/kanister/pkg/logsafe"
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

func kopiaS3Args(location map[string]string, assumeRoleDuration time.Duration, artifactPrefix string) (logsafe.Cmd, error) {
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
