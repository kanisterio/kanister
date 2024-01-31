package cli

import "reflect"

// The global arguments for Kopia CLI.

// CommonArgs defines the common arguments for Kopia CLI.
type CommonArgs struct {
	ConfigFilePath string // ConfigFilePath is the path to the config file.
	LogDirectory   string // LogDirectory is the directory where logs are stored.
	LogLevel       string // LogLevel is the level of logging.
	RepoPassword   string // RepoPassword is the password for the repository.
}

// IsZero returns true if all CommonArgs fields have a zero value.
func (c CommonArgs) IsZero() bool {
	return isZero(c)
}

// CacheArgs defines the cache arguments for Kopia CLI.
type CacheArgs struct {
	CacheDirectory           string // CacheDirectory is the directory where cache is stored.
	ContentCacheSizeMB       int    // ContentCacheSizeMB is the size of the content cache in MB.
	ContentCacheSizeLimitMB  int    // ContentCacheSizeLimitMB is the maximum size of the content cache in MB.
	MetadataCacheSizeMB      int    // MetadataCacheSizeMB is the size of the metadata cache in MB.
	MetadataCacheSizeLimitMB int    // MetadataCacheSizeLimitMB is the maximum size of the metadata cache in MB.
}

// IsZero returns true if all CacheArgs fields have a zero value.
func (c CacheArgs) IsZero() bool {
	return isZero(c)
}

func isZero(s interface{}) bool {
	val := reflect.ValueOf(s)
	for i := 0; i < val.NumField(); i++ {
		if !val.Field(i).IsZero() {
			return false
		}
	}
	return true
}
