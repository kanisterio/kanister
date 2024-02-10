package policy

import "github.com/kanisterio/safecli"

const (
	DefaultCompressionAlgorithm = "s2-default"
)

// CompressionAlgorithm defines the compression algorithm to use.
type CompressionAlgorithm struct {
	CompressionAlgorithm string
}

func (f CompressionAlgorithm) value() string {
	if f.CompressionAlgorithm == "" {
		return DefaultCompressionAlgorithm
	}
	return f.CompressionAlgorithm
}

func (f CompressionAlgorithm) Apply(cli safecli.CommandAppender) error {
	cli.AppendLoggableKV("--compression", f.value())
	return nil
}

// CompressionPolicy defines the compression policy to use.
type CompressionPolicy struct {
	CompressionAlgorithm
}
