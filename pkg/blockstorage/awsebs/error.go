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

package awsebs

import (
	"github.com/aws/aws-sdk-go/aws/awserr"
)

const (
	dryRunOperationCode  = "DryRunOperation"
	snapshotNotFoundCode = "InvalidSnapshot.NotFound"
	volumeNotFoundCode   = "InvalidVolume.NotFound"
)

func isDryRunErr(err error) bool {
	return isError(err, dryRunOperationCode)
}

func isSnapNotFoundErr(err error) bool {
	return isError(err, snapshotNotFoundCode)
}

func isError(err error, code string) bool {
	awsErr, ok := err.(awserr.Error)
	return ok && awsErr.Code() == code
}

func isVolNotFoundErr(err error) bool {
	return isError(err, volumeNotFoundCode)
}
