package command

// CacheArgs has fields that can be used to set
// cache settings for different kopia repository operations
type CacheArgs struct {
	ContentCacheLimitMB  int
	MetadataCacheLimitMB int
}
