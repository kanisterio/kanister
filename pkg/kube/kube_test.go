package kube

import (
	"testing"

	. "gopkg.in/check.v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

var emptyListOptions = v1.ListOptions{}

const defaultNamespace = "default"
