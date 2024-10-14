package kanx

import (
	"bytes"
	"encoding/json"

	"gopkg.in/check.v1"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

type LoggerSuite struct{}

var _ = check.Suite(&LoggerSuite{})

type Log struct {
	File     *string `json:"File,omitempty"`
	Function *string `json:"Function,omitempty"`
	Line     *int    `json:"Line,omitempty"`
	Level    *string `json:"level,omitempty"`
	Msg      *string `json:"msg,omitempty"`
	Time     *string `json:"time,omitempty"`
	Boo      *string `json:"boo,omitempty"`
}

func (s *LoggerSuite) TestLogger(c *check.C) {
	buf := bytes.NewBuffer(nil)
	msg := []byte("hello!")

	lw := newLogWriter(log.Info(), buf)

	n, err := lw.Write(msg)
	c.Assert(err, check.IsNil)
	c.Assert(n, check.Equals, len(msg))

	l := Log{}
	err = json.Unmarshal(buf.Bytes(), &l)
	c.Assert(err, check.IsNil)

	c.Assert(l.File, check.NotNil)
	c.Assert(l.Function, check.NotNil)
	c.Assert(l.Line, check.Not(check.Equals), 0)
	c.Assert(l.Level, check.NotNil)
	c.Assert(*l.Msg, check.Equals, string(msg))
	c.Assert(l.Time, check.NotNil)
	c.Assert(l.Boo, check.IsNil)

	buf.Reset()
	lw.SetFields(field.M{"boo": "far"})
	n, err = lw.Write(msg)
	c.Assert(err, check.IsNil)
	c.Assert(n, check.Equals, len(msg))

	l = Log{}
	err = json.Unmarshal(buf.Bytes(), &l)
	c.Assert(err, check.IsNil)

	c.Assert(l.File, check.NotNil)
	c.Assert(l.Function, check.NotNil)
	c.Assert(l.Line, check.Not(check.Equals), 0)
	c.Assert(l.Level, check.NotNil)
	c.Assert(*l.Msg, check.Equals, string(msg))
	c.Assert(l.Time, check.NotNil)
	c.Assert(*l.Boo, check.Equals, "far")
}
