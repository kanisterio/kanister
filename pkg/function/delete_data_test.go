package function

import (
	. "gopkg.in/check.v1"
)

type DeleteDataSuite struct {
}

var _ = Suite(&DeleteDataSuite{})

func (s *DeleteDataSuite) TestGetSnapshotIDFromTag(c *C) {
	for _, tc := range []struct {
		log      string
		expected string
		checker  Checker
	}{
		{log: `[{"time":"2019-03-28T17:35:15.146526-07:00","hostname":"MacBook-Pro.local","username":"abc","uid":501,"gid":20,"tags":["backup123"],"id":"7c0bfeb93dd5b390a6eaf8a386ec8cb86e4631f2d96400407b529b53d979536a","short_id":"7c0bfeb9"}]`, expected: "7c0bfeb9", checker: IsNil},
		{log: `[{"time":"2019-03-28T17:35:15.146526-07:00","hostname":"MacBook-Pro.local","username":"abc","uid":501,"gid":20,"tags":["backup123"],"id":"7c0bfeb93dd5b390a6eaf8a386ec8cb86e4631f2d96400407b529b53d979536a","short_id":"7c0bfeb9"},{"time":"2019-03-28T17:35:15.146526-07:00","hostname":"MacBook-Pro.local","username":"abc","uid":501,"gid":20,"tags":["backup123"],"id":"7c0bfeb93dd5b390a6eaf8a386ec8cb86e4631f2d96400407b529b53d979536a","short_id":"7c0bfeb9"}]`, expected: "", checker: NotNil},
		{log: `null`, expected: "", checker: NotNil},
	} {
		id, err := GetSnapshotIDFromLog(tc.log)
		c.Assert(err, tc.checker)
		c.Assert(id, Equals, tc.expected)

	}
}
