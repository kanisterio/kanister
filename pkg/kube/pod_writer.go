package kube

import (
	"context"
	"io"
	"path/filepath"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/format"
)

// PodWriter specifies Kubernetes Client and the other params needed for writing content to a file
type PodWriter struct {
	cli           kubernetes.Interface
	namespace     string
	path          string
	podName       string
	containerName string
}

// NewPodWriter returns a new PodWriter given Kubernetes Client, Namespace, path of file, name of pod and container
func NewPodWriter(cli kubernetes.Interface, namespace, path, podName, containerName string) *PodWriter {
	return &PodWriter{
		cli:           cli,
		namespace:     namespace,
		path:          filepath.Clean(path),
		podName:       podName,
		containerName: containerName,
	}
}

// Write will create a new file(if not present) and write the provided content to the file
func (p *PodWriter) Write(ctx context.Context, content io.Reader) error {
	cmd := []string{"sh", "-c", "cat - > " + p.path}
	stdout, stderr, err := Exec(p.cli, p.namespace, p.podName, p.containerName, cmd, content)
	format.Log(p.podName, p.containerName, stdout)
	format.Log(p.podName, p.containerName, stderr)
	return errors.Wrap(err, "Failed to write contents to file")
}

// Remove will delete the file created by Write() func
func (p *PodWriter) Remove(ctx context.Context) error {
	cmd := []string{"sh", "-c", "rm " + p.path}
	stdout, stderr, err := Exec(p.cli, p.namespace, p.podName, p.containerName, cmd, nil)
	format.Log(p.podName, p.containerName, stdout)
	format.Log(p.podName, p.containerName, stderr)
	return errors.Wrap(err, "Failed to delete file")
}
