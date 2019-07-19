package awsefs

import (
	awsefs "github.com/aws/aws-sdk-go/service/efs"

	kantags "github.com/kanisterio/kanister/pkg/blockstorage/tags"
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

func filterWithTags(descriptions []*awsefs.FileSystemDescription, tags map[string]string) []*awsefs.FileSystemDescription {
	result := make([]*awsefs.FileSystemDescription, 0)
	for i, desc := range descriptions {
		if kantags.IsSubset(convertFromEFSTags(desc.Tags), tags) {
			result = append(result, descriptions[i])
		}
	}
	return result
}
