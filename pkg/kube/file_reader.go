package kube

import (
	"context"
	"path"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

type PodFileReader struct {
	cli       kubernetes.Interface
	podName   string
	namespace string
	container string
}

func NewPodFileReader(cli kubernetes.Interface, podName, namespace, container string) *PodFileReader {
	return &PodFileReader{
		cli:       cli,
		podName:   podName,
		namespace: namespace,
		container: container,
	}
}

func (r *PodFileReader) ReadFile(ctx context.Context, path string) (string, error) {
	cmd := []string{"sh", "-c", "cat " + path}
	stdout, stderr, err := Exec(r.cli, r.namespace, r.podName, r.container, cmd, nil)
	if err != nil {
		if stderr != "" {
			log.Print("Error executing command", field.M{"stderr": stderr})
		}
		return "", errors.Wrap(err, "Failed to write contents to file")
	}
	return stdout, nil
}

func (r *PodFileReader) ReadDir(ctx context.Context, dirPath string) (map[string]string, error) {
	cmd := []string{"sh", "-c", "ls -1 " + dirPath}
	stdout, stderr, err := Exec(r.cli, r.namespace, r.podName, r.container, cmd, nil)
	if err != nil {
		if stderr != "" {
			log.Print("Error executing command", field.M{"stderr": stderr})
		}
		return nil, errors.Wrap(err, "Failed to list files of directory")
	}
	op := map[string]string{}
	data := strings.Split(stdout, "\n")
	for _, file := range data {
		out, err := r.ReadFile(ctx, path.Join(dirPath, file))
		if err != nil {
			return nil, errors.Wrap(err, "Failed to read contents of file")
		}
		op[file] = out
	}
	return op, nil
}
