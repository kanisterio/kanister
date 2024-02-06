package command

// Repository storage sub commands.
var (
	FileSystem = Command{"filesystem"}
	GCS        = Command{"gcs"}
	Azure      = Command{"azure"}
	S3         = Command{"s3"}
)
