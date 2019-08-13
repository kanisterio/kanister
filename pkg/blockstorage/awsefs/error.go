package awsefs

import (
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/backup"
	awsefs "github.com/aws/aws-sdk-go/service/efs"
	"github.com/pkg/errors"
)

func isVolumeNotFound(err error) bool {
	switch errV := err.(type) {
	case awserr.Error:
		return errV.Code() == awsefs.ErrCodeFileSystemNotFound
	case errors.Causer:
		return isVolumeNotFound(errV.Cause())
	default:
		return false
	}
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

func isMountTargetNotFound(err error) bool {
	if awsErr, ok := err.(awserr.Error); ok {
		return awsErr.Code() == awsefs.ErrCodeMountTargetNotFound
	}
	return false
}
