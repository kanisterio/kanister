package location

import (
	"bytes"
	"context"
	"io"
	"math/rand"
	"testing"
	"time"

	. "gopkg.in/check.v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/objectstore"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/testutil"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type LocationSuite struct {
	osType         objectstore.ProviderType
	provider       objectstore.Provider
	rand           *rand.Rand
	root           objectstore.Bucket // root of the default test bucket
	suiteDirPrefix string             // directory name prefix for all tests in this suite
	testpath       string
	region         string // bucket region
	profile        param.Profile
}

const (
	testBucketName = "kanister-gcp-tests"
	testRegionS3   = "us-west-2"
)

var _ = Suite(&LocationSuite{osType: objectstore.ProviderTypeGCS, region: ""})

func (s *LocationSuite) SetUpSuite(c *C) {
	switch s.osType {
	case objectstore.ProviderTypeGCS:
		testutil.GetEnvOrSkip(c, "GOOGLE_APPLICATION_CREDENTIALS")
		location := crv1alpha1.Location{
			Type:   crv1alpha1.LocationTypeGCS,
			Bucket: testBucketName,
		}
		s.profile = *testutil.ObjectStoreProfileOrSkip(c, objectstore.ProviderTypeGCS, location)
	default:
		c.Fatalf("Unrecognized objectstore '%s'", s.osType)
	}
	var err error
	ctx := context.Background()

	s.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	pc := objectstore.ProviderConfig{Type: s.osType}
	secret, err := getOSSecret(s.osType, s.profile.Credential)
	c.Check(err, IsNil)
	s.provider, err = objectstore.NewProvider(ctx, pc, secret)
	c.Check(err, IsNil)
	c.Assert(s.provider, NotNil)

	s.root, err = objectstore.GetOrCreateBucket(ctx, s.provider, testBucketName, s.region)
	c.Check(err, IsNil)
	c.Assert(s.root, NotNil)
	s.suiteDirPrefix = time.Now().UTC().Format(time.RFC3339Nano)
	s.testpath = s.suiteDirPrefix + "/testlocation.txt"
}

func (s *LocationSuite) TearDownTest(c *C) {
	if s.testpath != "" {
		c.Assert(s.root, NotNil)
		ctx := context.Background()
		err := s.root.Delete(ctx, s.testpath)
		if err != nil {
			c.Log("Cannot cleanup test directory: ", s.testpath)
			return
		}
		err = s.provider.DeleteBucket(ctx, testBucketName)
		c.Check(err, IsNil)
	}
}

func (s *LocationSuite) TestWrite(c *C) {
	ctx := context.Background()
	for _, tc := range []struct {
		in      io.Reader
		bin     string
		args    []string
		env     []string
		checker Checker
	}{
		{
			in:      bytes.NewBufferString("hello"),
			bin:     "",
			args:    nil,
			env:     nil,
			checker: NotNil,
		},
		{
			in:      bytes.NewBufferString("hello"),
			bin:     "cat",
			args:    nil,
			env:     nil,
			checker: IsNil,
		},
		{
			in:      bytes.NewBufferString("echo hello"),
			bin:     "bash",
			args:    nil,
			env:     nil,
			checker: IsNil,
		},
		{
			in:      bytes.NewBufferString("INVALID"),
			bin:     "bash",
			args:    nil,
			env:     nil,
			checker: NotNil,
		},
	} {

		err := writeExec(ctx, tc.in, tc.bin, tc.args, tc.env)
		c.Check(err, tc.checker)
	}
}

func (s *LocationSuite) TestRead(c *C) {
	ctx := context.Background()
	for _, tc := range []struct {
		out     string
		bin     string
		args    []string
		env     []string
		checker Checker
	}{
		{
			out:     "",
			bin:     "",
			args:    nil,
			env:     nil,
			checker: NotNil,
		},
		{
			out:     "",
			bin:     "echo",
			args:    []string{"-n"},
			env:     nil,
			checker: IsNil,
		},
		{
			out:     "hello",
			bin:     "echo",
			args:    []string{"-n", "hello"},
			env:     nil,
			checker: IsNil,
		},
	} {
		buf := bytes.NewBuffer(nil)
		err := readExec(ctx, buf, tc.bin, tc.args, tc.env)
		c.Check(err, tc.checker)
		c.Check(buf.String(), Equals, tc.out)
	}
}

func (s *LocationSuite) TestWriteAndReadData(c *C) {
	ctx := context.Background()
	teststring := "test-content"
	err := writeData(ctx, s.osType, s.profile, bytes.NewBufferString(teststring), s.testpath)
	c.Check(err, IsNil)
	buf := bytes.NewBuffer(nil)
	err = readData(ctx, s.osType, s.profile, buf, s.testpath)
	c.Check(err, IsNil)
	c.Check(buf.String(), Equals, teststring)

}
