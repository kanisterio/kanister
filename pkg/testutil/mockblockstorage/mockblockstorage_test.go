package mockblockstorage

import (
	"testing"

	. "gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/blockstorage"
)

func Test(t *testing.T) { TestingT(t) }

type MockSuite struct{}

var _ = Suite(&MockSuite{})

func (s *MockSuite) TestMockStorage(c *C) {
	mock := Get(blockstorage.TypeEBS)
	c.Assert(mock.Type(), Equals, blockstorage.TypeEBS)
}
