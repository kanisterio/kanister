package model

import (
	"strconv"
	"strings"

	rs "github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
)

// Location is a map of key-value pairs that represent different storage properties.
type Location map[string][]byte

// Type returns the location type.
func (l Location) Type() rs.LocType {
	return rs.LocType(string(l[rs.TypeKey]))
}

// Region returns the location region.
func (l Location) Region() string {
	return string(l[rs.RegionKey])
}

// BucketName returns the location bucket name.
func (l Location) BucketName() string {
	return string(l[rs.BucketKey])
}

// Endpoint returns the location endpoint.
func (l Location) Endpoint() string {
	return string(l[rs.EndpointKey])
}

// Prefix returns the location prefix.
func (l Location) Prefix() string {
	return string(l[rs.PrefixKey])
}

// IsInsecureEndpoint returns true if the location endpoint is insecure/http.
func (l Location) IsInsecureEndpoint() bool {
	return strings.HasPrefix(l.Endpoint(), "http:")
}

// HasSkipSSLVerify returns true if the location has skip SSL verification.
func (l Location) HasSkipSSLVerify() bool {
	v, _ := strconv.ParseBool(string(l[rs.SkipSSLVerifyKey]))
	return v
}
