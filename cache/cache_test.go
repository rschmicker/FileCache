package cache

import (
	. "gopkg.in/check.v1"
	"path/filepath"
	"testing"
)

type CacheTestSuite struct{}

var _ = Suite(&CacheTestSuite{})

func Test(t *testing.T) { TestingT(t) }

func (s *CacheTestSuite) TestInitialize(c *C) {
	ct := &CacheTable{}
	ct.RamDiskPath = "/dev/shm/cache/"
	one, _ := filepath.Abs("./cache.go")
	two, _ := filepath.Abs("./cache_test.go")
	ct.Files = []string{one, two}
	ct.Initialize()
	c.Check(len(ct.Table), Equals, 2)
}
