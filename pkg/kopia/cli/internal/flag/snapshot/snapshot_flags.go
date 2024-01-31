package snapshot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/kopia/cli"
	"github.com/kanisterio/kanister/pkg/kopia/cli/internal/flag"
	"github.com/kanisterio/kanister/pkg/utils"
)

const (
	DefaultProgressUpdateInterval = "1h"

	numOfPartsInTag = 2
)

// Parallel creates a new parallel flag with a given parallelism.
func Parallel(parallelism int) flag.Applier {
	p := strconv.Itoa(parallelism)
	return flag.NewStringFlag("--parallel", p)
}

// ProgressUpdateInterval creates a new progress update interval flag with a given interval.
// If interval is less than or equal to 0, the DefaultProgressUpdateInterval value is used.
func ProgressUpdateInterval(interval time.Duration) flag.Applier {
	value := DefaultProgressUpdateInterval
	if interval > 0 {
		value = utils.DurationToString(utils.RoundUpDuration(interval))
	}
	return flag.NewStringFlag("--progress-update-interval", value)
}

// PathToBackup creates a new path to backup argument with a given path.
// If the path is empty, it returns ErrInvalidBackupPath.
func PathToBackup(path string) flag.Applier {
	if path == "" {
		return flag.ErrorFlag(cli.ErrInvalidBackupPath)
	}
	return flag.NewStringValue(path)
}

// validateTags validates all tags and returns an error if any of the tags are invalid.
func validateTags(tags []string) error {
	for _, tag := range tags {
		if err := validateTag(tag); err != nil {
			return err
		}
	}
	return nil
}

// validateTag validates a tag and returns an error if the tag is invalid.
func validateTag(tag string) error {
	if tag == "" {
		return errors.Wrapf(cli.ErrInvalidTag, "tag cannot be empty")
	}

	parts := strings.SplitN(tag, ":", numOfPartsInTag)
	if len(parts) != numOfPartsInTag {
		return errors.Wrapf(cli.ErrInvalidTag, fmt.Sprintf("requires <key>:<value> but got %q", tag))
	}

	return nil
}

// newTags creates a new newTags flag with a given newTags.
// If validate is true, it validates the tags and returns
// ErrInvalidTag error if any of the tags are invalid.
func newTags(tags []string, validate bool) flag.Applier {
	if validate {
		if err := validateTags(tags); err != nil {
			return flag.ErrorFlag(err)
		}
	}

	var flags []flag.Applier
	for _, tag := range tags {
		flags = append(flags, flag.NewStringFlag("--tags", tag))
	}
	return flag.NewFlags(flags...)
}

// Tags creates a new tags flag with the given tags.
// If any of the tags are invalid, it returns ErrInvalidTag.
func Tags(tags []string) flag.Applier {
	return newTags(tags, true)
}

// TagsWithNoValidation creates a new tags flag with the given tags,
// even if some of the tags are invalid.
func TagsWithNoValidation(tags []string) flag.Applier {
	return newTags(tags, false)
}
