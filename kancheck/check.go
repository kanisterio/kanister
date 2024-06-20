package check

import (
	"testing"

	checkv1 "github.com/kastenhq/check"
)

//nolint:gochecknoinits // testing flags must be registered in an init function
func init() {
	testing.Init()
}

type (
	C           = checkv1.C
	Checker     = checkv1.Checker
	CheckerInfo = checkv1.CheckerInfo
)

var (
	FitsTypeOf   = checkv1.FitsTypeOf
	DeepEquals   = checkv1.DeepEquals
	TestingT     = checkv1.TestingT
	Suite        = checkv1.Suite
	Commentf     = checkv1.Commentf
	Equals       = checkv1.Equals
	ErrorMatches = checkv1.ErrorMatches
	HasLen       = checkv1.HasLen
	Implements   = checkv1.Implements
	IsNil        = checkv1.IsNil
	NotNil       = checkv1.NotNil
	Matches      = checkv1.Matches
	Not          = checkv1.Not
	PanicMatches = checkv1.PanicMatches
)
