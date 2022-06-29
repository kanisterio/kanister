package kopia

import (
	"fmt"
	"strings"

	"gopkg.in/check.v1"
	v1 "k8s.io/api/core/v1"

	"github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/secrets"
)

type KopiaRepositorySuite struct{}

var _ = check.Suite(&KopiaRepositorySuite{})

func (s *KopiaRepositorySuite) TestKopiaBlobStoreArgs(c *check.C) {
	for _, tc := range []struct {
		testCase    string
		prof        profile
		prefix      string
		check       check.Checker
		expectedCmd []string
		expectedLog string
	}{
		{
			testCase: "Kanister S3 Profile with Endpoint, SkipSSLVerify and Cred type Secret",
			prof: &KanisterProfile{
				Profile: &param.Profile{
					Location: v1alpha1.Location{
						Type:     v1alpha1.LocationTypeS3Compliant,
						Endpoint: "endpoint",
						Bucket:   "my-bucket",
					},
					Credential: param.Credential{
						Type: param.CredentialTypeSecret,
						Secret: &v1.Secret{
							Type: v1.SecretType(secrets.AWSSecretType),
							Data: map[string][]byte{
								secrets.AWSAccessKeyID:     []byte("id"),
								secrets.AWSSecretAccessKey: []byte("secret"),
							},
						},
					},
					SkipSSLVerify: true,
				},
			},
			prefix: "my-prefix/",
			check:  check.IsNil,
			expectedCmd: []string{
				"s3",
				"--bucket=my-bucket",
				"--endpoint=endpoint",
				"--access-key=id",
				"--secret-access-key=secret",
				"--prefix=my-prefix/",
				"--disable-tls-verification",
			},
			expectedLog: "s3 --bucket=my-bucket --endpoint=endpoint --access-key=<****> --secret-access-key=<****> --prefix=my-prefix/ --disable-tls-verification",
		},
		{
			testCase: "Kanister S3 Profile wiht Endpoint and Cred type KeyPair",
			prof: &KanisterProfile{
				Profile: &param.Profile{
					Location: v1alpha1.Location{
						Type:     v1alpha1.LocationTypeS3Compliant,
						Endpoint: "endpoint",
						Bucket:   "my-bucket",
					},
					Credential: param.Credential{
						Type: param.CredentialTypeKeyPair,
						KeyPair: &param.KeyPair{
							ID:     "id",
							Secret: "secret",
						},
					},
				},
			},
			prefix: "my-prefix",
			check:  check.IsNil,
			expectedCmd: []string{
				"s3",
				"--bucket=my-bucket",
				"--endpoint=endpoint",
				"--access-key=id",
				"--secret-access-key=secret",
				"--prefix=my-prefix",
			},
			expectedLog: "s3 --bucket=my-bucket --endpoint=endpoint --access-key=<****> --secret-access-key=<****> --prefix=my-prefix",
		},
		{
			testCase: "Kanister S3 Profile with Endpoint, SkipSSLVerify false and Cred type Secret",
			prof: &KanisterProfile{
				Profile: &param.Profile{
					Location: v1alpha1.Location{
						Type:     v1alpha1.LocationTypeS3Compliant,
						Endpoint: "endpoint/", // Remove trailing slash
						Bucket:   "my-bucket",
					},
					Credential: param.Credential{
						Type: param.CredentialTypeSecret,
						Secret: &v1.Secret{
							Type: v1.SecretType(secrets.AWSSecretType),
							Data: map[string][]byte{
								secrets.AWSAccessKeyID:     []byte("id"),
								secrets.AWSSecretAccessKey: []byte("secret"),
							},
						},
					},
					SkipSSLVerify: false,
				},
			},
			prefix: "my-prefix",
			check:  check.IsNil,
			expectedCmd: []string{
				"s3",
				"--bucket=my-bucket",
				"--endpoint=endpoint",
				"--access-key=id",
				"--secret-access-key=secret",
				"--prefix=my-prefix",
			},
			expectedLog: "s3 --bucket=my-bucket --endpoint=endpoint --access-key=<****> --secret-access-key=<****> --prefix=my-prefix",
		},
		{
			testCase: "Kanister S3 Profile with Endpoint with trailing slashes, SkipSSLVerify and Cred type Secret",
			prof: &KanisterProfile{
				Profile: &param.Profile{
					Location: v1alpha1.Location{
						Type:     v1alpha1.LocationTypeS3Compliant,
						Endpoint: "endpoint/////////", // Also remove all of the trailing slashes
						Bucket:   "my-bucket",
					},
					Credential: param.Credential{
						Type: param.CredentialTypeSecret,
						Secret: &v1.Secret{
							Type: v1.SecretType(secrets.AWSSecretType),
							Data: map[string][]byte{
								secrets.AWSAccessKeyID:     []byte("id"),
								secrets.AWSSecretAccessKey: []byte("secret"),
							},
						},
					},
					SkipSSLVerify: true,
				},
			},
			prefix: "my-prefix",
			check:  check.IsNil,
			expectedCmd: []string{
				"s3",
				"--bucket=my-bucket",
				"--endpoint=endpoint",
				"--access-key=id",
				"--secret-access-key=secret",
				"--prefix=my-prefix",
				"--disable-tls-verification",
			},
			expectedLog: "s3 --bucket=my-bucket --endpoint=endpoint --access-key=<****> --secret-access-key=<****> --prefix=my-prefix --disable-tls-verification",
		},
		{
			testCase: "Kanister GCS Profile",
			prof: &KanisterProfile{
				Profile: &param.Profile{
					Location: v1alpha1.Location{
						Type:   v1alpha1.LocationTypeGCS,
						Bucket: "my-bucket",
					},
					Credential: param.Credential{
						Type: param.CredentialTypeSecret,
						Secret: &v1.Secret{
							Type: v1.SecretType(secrets.AWSSecretType),
							Data: map[string][]byte{
								secrets.AWSAccessKeyID:     []byte("id"),
								secrets.AWSSecretAccessKey: []byte("secret"),
							},
						},
					},
					SkipSSLVerify: false,
				},
			},
			prefix: "my-prefix/",
			check:  check.IsNil,
			expectedCmd: []string{
				"gcs",
				"--bucket=my-bucket",
				fmt.Sprintf("--credentials-file=%s", consts.GoogleCloudCredsFilePath),
				"--prefix=my-prefix/",
			},
			expectedLog: "gcs --bucket=my-bucket --credentials-file=/tmp/creds.txt --prefix=my-prefix/",
		},
		{
			testCase: "Kanister S3 Profile with Endpoint, Prefix, SkipSSLVerify and Cred type Secret",
			prof: &KanisterProfile{
				Profile: &param.Profile{
					Location: v1alpha1.Location{
						Type:     v1alpha1.LocationTypeS3Compliant,
						Endpoint: "endpoint",
						Bucket:   "my-bucket",
						Prefix:   "cluster-id",
					},
					Credential: param.Credential{
						Type: param.CredentialTypeSecret,
						Secret: &v1.Secret{
							Type: v1.SecretType(secrets.AWSSecretType),
							Data: map[string][]byte{
								secrets.AWSAccessKeyID:     []byte("id"),
								secrets.AWSSecretAccessKey: []byte("secret"),
							},
						},
					},
					SkipSSLVerify: true,
				},
			},
			prefix: "my-prefix/",
			check:  check.IsNil,
			expectedCmd: []string{
				"s3",
				"--bucket=my-bucket",
				"--endpoint=endpoint",
				"--access-key=id",
				"--secret-access-key=secret",
				"--prefix=cluster-id/my-prefix/",
				"--disable-tls-verification",
			},
			expectedLog: "s3 --bucket=my-bucket --endpoint=endpoint --access-key=<****> --secret-access-key=<****> --prefix=cluster-id/my-prefix/ --disable-tls-verification",
		},
		{
			testCase: "Kanister GCS Profile with Prefix",
			prof: &KanisterProfile{
				Profile: &param.Profile{
					Location: v1alpha1.Location{
						Type:   v1alpha1.LocationTypeGCS,
						Bucket: "my-bucket",
						Prefix: "cluster-id",
					},
					Credential: param.Credential{
						Type: param.CredentialTypeSecret,
						Secret: &v1.Secret{
							Type: v1.SecretType(secrets.AWSSecretType),
							Data: map[string][]byte{
								secrets.AWSAccessKeyID:     []byte("id"),
								secrets.AWSSecretAccessKey: []byte("secret"),
							},
						},
					},
					SkipSSLVerify: false,
				},
			},
			prefix: "my-prefix/",
			check:  check.IsNil,
			expectedCmd: []string{
				"gcs",
				"--bucket=my-bucket",
				fmt.Sprintf("--credentials-file=%s", consts.GoogleCloudCredsFilePath),
				"--prefix=cluster-id/my-prefix/",
			},
			expectedLog: "gcs --bucket=my-bucket --credentials-file=/tmp/creds.txt --prefix=cluster-id/my-prefix/",
		},
		{
			testCase: "Kanister Azure Profile",
			prof: &KanisterProfile{
				Profile: &param.Profile{
					Location: v1alpha1.Location{
						Type:   v1alpha1.LocationTypeAzure,
						Bucket: "my-bucket",
					},
					Credential: param.Credential{
						Type: param.CredentialTypeKeyPair,
						KeyPair: &param.KeyPair{
							ID:     "id",
							Secret: "secret",
						},
					},
					SkipSSLVerify: false,
				},
			},
			prefix: "my-prefix",
			check:  check.IsNil,
			expectedCmd: []string{
				"azure",
				"--container=my-bucket",
				"--prefix=my-prefix",
				"--storage-account=id",
				"--storage-key=secret",
			},
			expectedLog: "azure --container=my-bucket --prefix=my-prefix --storage-account=<****> --storage-key=<****>",
		},
		{
			testCase: "Kanister Azure Profile with Prefix",
			prof: &KanisterProfile{
				Profile: &param.Profile{
					Location: v1alpha1.Location{
						Type:   v1alpha1.LocationTypeAzure,
						Bucket: "my-bucket",
						Prefix: "cluster-id",
					},
					Credential: param.Credential{
						Type: param.CredentialTypeKeyPair,
						KeyPair: &param.KeyPair{
							ID:     "id",
							Secret: "secret",
						},
					},
					SkipSSLVerify: false,
				},
			},
			prefix: "my-prefix/",
			check:  check.IsNil,
			expectedCmd: []string{
				"azure",
				"--container=my-bucket",
				"--prefix=cluster-id/my-prefix/",
				"--storage-account=id",
				"--storage-key=secret",
			},
			expectedLog: "azure --container=my-bucket --prefix=cluster-id/my-prefix/ --storage-account=<****> --storage-key=<****>",
		},
		{
			testCase: "Kanister Azure Profile with Cred type Secret",
			prof: &KanisterProfile{
				Profile: &param.Profile{
					Location: v1alpha1.Location{
						Type:   v1alpha1.LocationTypeAzure,
						Bucket: "my-bucket",
						Prefix: "cluster-id",
					},
					Credential: param.Credential{
						Type: param.CredentialTypeSecret,
						Secret: &v1.Secret{
							Type: v1.SecretType(secrets.AzureSecretType),
							Data: map[string][]byte{
								secrets.AzureStorageAccountID:   []byte("id"),
								secrets.AzureStorageAccountKey:  []byte("secret"),
								secrets.AzureStorageEnvironment: []byte("AzureUSGovernmentCloud"),
							},
						},
					},
				},
			},
			prefix: "my-prefix",
			check:  check.IsNil,
			expectedCmd: []string{
				"azure",
				"--container=my-bucket",
				"--prefix=cluster-id/my-prefix/",
				"--storage-account=id",
				"--storage-key=secret",
				"--storage-domain=blob.core.usgovcloudapi.net",
			},
			expectedLog: "azure --container=my-bucket --prefix=cluster-id/my-prefix/ --storage-account=<****> --storage-key=<****> --storage-domain=blob.core.usgovcloudapi.net",
		},
		{
			testCase: "Kanister S3 Profile with HTTPS Endpoint",
			prof: &KanisterProfile{
				Profile: &param.Profile{
					Location: v1alpha1.Location{
						Type:     v1alpha1.LocationTypeS3Compliant,
						Endpoint: "https://endpoint",
						Bucket:   "my-bucket",
					},
					Credential: param.Credential{
						Type: param.CredentialTypeSecret,
						Secret: &v1.Secret{
							Type: v1.SecretType(secrets.AWSSecretType),
							Data: map[string][]byte{
								secrets.AWSAccessKeyID:     []byte("id"),
								secrets.AWSSecretAccessKey: []byte("secret"),
							},
						},
					},
					SkipSSLVerify: false,
				},
			},
			prefix: "my-prefix",
			check:  check.IsNil,
			expectedCmd: []string{
				"s3",
				"--bucket=my-bucket",
				"--endpoint=endpoint",
				"--access-key=id",
				"--secret-access-key=secret",
				"--prefix=my-prefix",
			},
			expectedLog: "s3 --bucket=my-bucket --endpoint=endpoint --access-key=<****> --secret-access-key=<****> --prefix=my-prefix",
		},
		{
			testCase: "Kanister S3 Profile with HTTP Endpoint and SkipSSLVerify",
			prof: &KanisterProfile{
				Profile: &param.Profile{
					Location: v1alpha1.Location{
						Type:     v1alpha1.LocationTypeS3Compliant,
						Endpoint: "http://endpoint",
						Bucket:   "my-bucket",
					},
					Credential: param.Credential{
						Type: param.CredentialTypeSecret,
						Secret: &v1.Secret{
							Type: v1.SecretType(secrets.AWSSecretType),
							Data: map[string][]byte{
								secrets.AWSAccessKeyID:     []byte("id"),
								secrets.AWSSecretAccessKey: []byte("secret"),
							},
						},
					},
					SkipSSLVerify: true,
				},
			},
			prefix: "my-prefix",
			check:  check.IsNil,
			expectedCmd: []string{
				"s3",
				"--bucket=my-bucket",
				"--endpoint=endpoint",
				"--disable-tls",
				"--access-key=id",
				"--secret-access-key=secret",
				"--prefix=my-prefix",
				"--disable-tls-verification",
			},
			expectedLog: "s3 --bucket=my-bucket --endpoint=endpoint --disable-tls --access-key=<****> --secret-access-key=<****> --prefix=my-prefix --disable-tls-verification",
		},
	} {
		args, err := kopiaBlobStoreArgs(tc.prof, tc.prefix)
		c.Assert(err, tc.check, check.Commentf("testCase: %s", tc.testCase))
		c.Check(args.PlainText(), check.DeepEquals, strings.Join(tc.expectedCmd, " "), check.Commentf("testCase: %s", tc.testCase))
		c.Check(args.String(), check.Equals, tc.expectedLog)
	}
}
