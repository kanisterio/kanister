// Copyright 2019 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package awsefs

import (
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/backup"
	awsefs "github.com/aws/aws-sdk-go/service/efs"
	"github.com/pkg/errors"
)

func isVolumeNotFound(err error) bool {
	switch errV := errors.Cause(err).(type) {
	case awserr.Error:
		return errV.Code() == awsefs.ErrCodeFileSystemNotFound
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
