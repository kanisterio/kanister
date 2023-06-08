// +k8s:deepcopy-gen=package,register

// Package v1alpha1 is the v1alpha1 version of the API.
// +groupName=cr.kanister.io
// +versionName=v1alpha1
package v1alpha1

// While generating client files, we need code-generator package to be installed
// but this package is not used anywhere hence go.mod removes this from
// required package. hence added an empty import.
import _ "k8s.io/code-generator"
