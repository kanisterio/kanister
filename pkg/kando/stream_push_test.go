package kando

import (
	. "gopkg.in/check.v1"
)

type StreamPushTestSuite struct{}

var _ = Suite(&StreamPushTestSuite{})

func (s *StreamPushTestSuite) TestEnclosePassword(c *C) {
	for _, tc := range []struct {
		input, output string
	}{
		// we dont handle back ticks yet
		{
			input:  "this-is3543%$%#$#()*&)~~`-dummy-pass4534",
			output: "\"this-is3543%$%#$#()*&)~~\\`-dummy-pass4534\"",
		},
		{
			input:  "12345",
			output: `"12345"`,
		},
		{
			input:  "this-is-dummy-pass",
			output: "\"this-is-dummy-pass\"",
		},
		{
			input:  "",
			output: "",
		},
		{
			input:  " !\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~",
			output: "\" !\\\"#$%&'()*+,-./:;<=>?@[\\]^_\\`{|}~\"",
		},
		{
			input:  " with\"double double \"quote",
			output: "\" with\\\"double double \\\"quote\"", // == " with\"double double \"quote"
		},
		{
			input:  " this'is `another\" input",
			output: "\" this'is \\`another\\\" input\"",
		},
	} {
		output := enclosePassword(tc.input)
		c.Assert(output, Equals, tc.output)
	}
}
