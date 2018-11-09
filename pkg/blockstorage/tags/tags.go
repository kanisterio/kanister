package tags

import (
	"os"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	// ClusterTagKey is used to tag resources with the cluster name
	ClusterTagKey = "kanister.io/clustername"
	// VersionTagKey is used to tag resources with the K10 version
	VersionTagKey = "kanister.io/version"
	// AppNameTag is used to tag volumes with the app they belong to
	AppNameTag = "kanister.io/appname"
)

// GetTags returns the tags to set on a resource
func GetTags(inputTags map[string]string) map[string]string {
	tags := GetStdTags()

	// inputTags could've be derived from an existing object so only add tags that are
	// missing (ignore ones that already exist)
	return AddMissingTags(tags, inputTags)
}

// GetStdTags returns a set of standard tags to use for tagging resources
func GetStdTags() map[string]string {
	version := os.Getenv("VERSION")
	clustername := os.Getenv("CLUSTER_NAME")

	stdTags := map[string]string{
		ClusterTagKey: clustername,
		VersionTagKey: version,
	}
	return stdTags
}

// AddMissingTags returns a new map which contains 'existing' + any tags
// in 'tagsToAdd' that did not exist
func AddMissingTags(existingTags map[string]string, tagsToAdd map[string]string) map[string]string {
	ret := make(map[string]string, len(existingTags))
	for k, v := range existingTags {
		ret[k] = v
	}
	// Add missing tags
	for k, v := range tagsToAdd {
		if val, ok := ret[k]; ok {
			log.Infof("Ignoring duplicate tag: %s:%s. Retained value: %s", k, v, val)
		} else {
			ret[k] = v
		}
	}
	return ret
}

// SanitizeValueForGCP shrink value if needed and change prohibited chars
func SanitizeValueForGCP(value string) string {
	// From https://cloud.google.com/compute/docs/labeling-resources
	// - Keys and values cannot be longer than 63 characters each.
	// - Keys and values can only contain lowercase letters, numeric
	//   characters, underscores, and dashes. International characters
	//   are allowed.
	// - Label keys must start with a lowercase letter and international
	//   characters are allowed.
	re := regexp.MustCompile("[^a-z0-9_-]")
	sanitizedVal := value
	if len(sanitizedVal) > 63 {
		sanitizedVal = sanitizedVal[0:63]
	}
	sanitizedVal = strings.ToLower(sanitizedVal)
	sanitizedVal = re.ReplaceAllString(sanitizedVal, "_")
	return sanitizedVal
}
