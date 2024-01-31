package s3

import (
	"testing"

	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/test"
	"gopkg.in/check.v1"
)

func TestStorageS3Flags(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(test.NewFlagSuite([]test.FlagTest{
	{
		Name: "Empty Bucket should not generate a flag",
		Flag: Bucket(""),
	},
	{
		Name:        "Bucket with value should generate a flag with the given value",
		Flag:        Bucket("bucket"),
		ExpectedCLI: []string{"--bucket=bucket"},
	},
	{
		Name: "Empty Endpoint should not generate a flag",
		Flag: Endpoint(""),
	},
	{
		Name:        "Endpoint with value should generate a flag with the given value",
		Flag:        Endpoint("endpoint"),
		ExpectedCLI: []string{"--endpoint=endpoint"},
	},
	{
		Name: "Empty Prefix should not generate a flag",
		Flag: Prefix(""),
	},
	{
		Name:        "Prefix with value should generate a flag with the given value",
		Flag:        Prefix("prefix"),
		ExpectedCLI: []string{"--prefix=prefix"},
	},
	{
		Name: "Empty Region should not generate a flag",
		Flag: Region(""),
	},
	{
		Name:        "Region with value should generate a flag with the given value",
		Flag:        Region("region"),
		ExpectedCLI: []string{"--region=region"},
	},
	{
		Name: "DisableTLS(false) should not generate a flag",
		Flag: DisableTLS(false),
	},
	{
		Name:        "DisableTLS(true) should generate a flag",
		Flag:        DisableTLS(true),
		ExpectedCLI: []string{"--disable-tls"},
	},
	{
		Name: "DisableTLSVerify(false) should not generate a flag",
		Flag: DisableTLSVerify(false),
	},
	{
		Name:        "DisableTLSVerify(true) should generate a flag",
		Flag:        DisableTLSVerify(true),
		ExpectedCLI: []string{"--disable-tls-verification"},
	},
}))
