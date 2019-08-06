package chronicle

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"

	"github.com/kanisterio/kanister/pkg/location"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/pkg/errors"
)

func Pull(ctx context.Context, target io.Writer, p param.Profile, manifest string) error {
	// Read manifest
	buf := bytes.NewBuffer(nil)
	location.Read(ctx, buf, p, manifest)
	// Read Data
	data, err := ioutil.ReadAll(buf)
	if err != nil {
		return errors.Wrap(err, "Could not read chronicle manifest")
	}
	return location.Read(ctx, target, p, string(data))
}
