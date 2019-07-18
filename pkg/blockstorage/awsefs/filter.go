package awsefs

import (
	awsefs "github.com/aws/aws-sdk-go/service/efs"
)

func filterAvailable(descriptions []*awsefs.FileSystemDescription) []*awsefs.FileSystemDescription {
	result := make([]*awsefs.FileSystemDescription, 0)
	for _, desc := range descriptions {
		if *desc.LifeCycleState == awsefs.LifeCycleStateAvailable {
			result = append(result, desc)
		}
	}
	return result
}
