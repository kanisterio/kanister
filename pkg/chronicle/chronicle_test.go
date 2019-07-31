package chronicle

import (
	"context"
	"path/filepath"
	"testing"

	. "gopkg.in/check.v1"
	"k8s.io/apimachinery/pkg/util/rand"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/objectstore"
	"github.com/kanisterio/kanister/pkg/testutil"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type ChronicleSuite struct{}

var _ = Suite(&ChronicleSuite{})

func (s *ChronicleSuite) TestPush(c *C) {
	osType := objectstore.ProviderTypeS3
	loc := crv1alpha1.Location{
		Type:   crv1alpha1.LocationTypeS3Compliant,
		Region: testutil.TestS3Region,
		Bucket: testutil.TestS3BucketName,
	}
	prof := *testutil.ObjectStoreProfileOrSkip(c, osType, loc)
	pp := filepath.Join(c.MkDir(), "profile.json")
	err := writeProfile(pp, prof)
	c.Assert(err, IsNil)

	p := PushParams{
		ProfilePath:  pp,
		ArtifactPath: rand.String(10),
		Command:      []string{"echo hello"},
	}
	ctx := context.Background()
	err = push(ctx, p, 0)
	c.Assert(err, IsNil)
}
