package kanx

import (
	"bytes"
	"encoding/json"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
	. "gopkg.in/check.v1"
)

type LoggerSuite struct{}

var _ = Suite(&LoggerSuite{})

type Log struct {
	File     *string `json:"File,omitempty"`
	Function *string `json:"Function,omitempty"`
	Line     *int    `json:"Line,omitempty"`
	Level    *string `json:"level,omitempty"`
	Msg      *string `json:"msg,omitempty"`
	Time     *string `json:"time,omitempty"`
	Boo      *string `json:"boo,omitempty"`
}

func (s *LoggerSuite) TestLogger(c *C) {
	buf := bytes.NewBuffer(nil)
	msg := []byte("hello!")

	lw := newLogWriter(log.Info(), buf)

	n, err := lw.Write(msg)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, len(msg))

	l := Log{}
	err = json.Unmarshal(buf.Bytes(), &l)
	c.Assert(err, IsNil)

	c.Assert(l.File, NotNil)
	c.Assert(l.Function, NotNil)
	c.Assert(l.Line, Not(Equals), 0)
	c.Assert(l.Level, NotNil)
	c.Assert(*l.Msg, Equals, string(msg))
	c.Assert(l.Time, NotNil)
	c.Assert(l.Boo, IsNil)

	buf.Reset()
	lw.SetFields(field.M{"boo": "far"})
	n, err = lw.Write(msg)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, len(msg))

	l = Log{}
	err = json.Unmarshal(buf.Bytes(), &l)
	c.Assert(err, IsNil)

	c.Assert(l.File, NotNil)
	c.Assert(l.Function, NotNil)
	c.Assert(l.Line, Not(Equals), 0)
	c.Assert(l.Level, NotNil)
	c.Assert(*l.Msg, Equals, string(msg))
	c.Assert(l.Time, NotNil)
	c.Assert(*l.Boo, Equals, "far")
}
