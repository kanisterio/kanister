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
