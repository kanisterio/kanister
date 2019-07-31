package chronicle

import (
	"time"
)

type PushParams struct {
	ProfilePath  string
	ArtifactPath string
	Frequency    time.Duration
	EnvDir       string
	Command      []string
}

func (p PushParams) Validate() error {
	return nil
}

func Push(p PushParams) error {
	return nil
}
