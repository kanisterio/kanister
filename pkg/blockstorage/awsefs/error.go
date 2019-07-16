package awsefs

import (
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/backup"
	awsefs "github.com/aws/aws-sdk-go/service/efs"
)

func isVolumeNotFound(err error) bool {
	if awsErr, ok := err.(awserr.Error); ok {
		return awsErr.Code() == awsefs.ErrCodeFileSystemNotFound
	}
	return false
}

func isBackupVaultAlreadyExists(err error) bool {
	if awsErr, ok := err.(awserr.Error); ok {
		return awsErr.Code() == backup.ErrCodeAlreadyExistsException
	}
	return false
}

func isRecoveryPointNotFound(err error) bool {
	if awsErr, ok := err.(awserr.Error); ok {
		return awsErr.Code() == backup.ErrCodeResourceNotFoundException
	}
	return false
}
