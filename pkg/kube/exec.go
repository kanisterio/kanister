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

package kube

import (
	"bytes"
	"context"
	"io"
	"net/url"
	"strings"

	"github.com/kanisterio/kanister/pkg/format"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// ExecOptions passed to ExecWithOptions
type ExecOptions struct {
	Command []string

	Namespace     string
	PodName       string
	ContainerName string

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// Exec is our version of the call to `kubectl exec` that does not depend on
// k8s.io/kubernetes.
func Exec(ctx context.Context, cli kubernetes.Interface, namespace, pod, container string, command []string, stdin io.Reader) (string, string, error) {
	opts := ExecOptions{
		Command:       command,
		Namespace:     namespace,
		PodName:       pod,
		ContainerName: container,
		Stdin:         stdin,
	}
	return ExecWithOptions(ctx, cli, opts)
}

// ExecOutput is similar to Exec, except that inbound outputs are written to the
// provided stdout and stderr. Unlike Exec, the outputs are not returned to the
// caller.
func ExecOutput(ctx context.Context, cli kubernetes.Interface, namespace, pod, container string, command []string, stdin io.Reader, stdout, stderr io.Writer) error {
	opts := ExecOptions{
		Command:       command,
		Namespace:     namespace,
		PodName:       pod,
		ContainerName: container,
		Stdin:         stdin,
		Stdout: &format.Writer{
			W:         stdout,
			Pod:       pod,
			Container: container,
		},
		Stderr: &format.Writer{
			W:         stderr,
			Pod:       pod,
			Container: container,
		},
	}

	_, _, err := ExecWithOptions(ctx, cli, opts)
	return err
}

// ExecWithOptions executes a command in the specified container,
// returning stdout, stderr and error. `options` allowed for
// additional parameters to be passed.
func ExecWithOptions(ctx context.Context, kubeCli kubernetes.Interface, options ExecOptions) (string, string, error) {
	config, err := LoadConfig()
	if err != nil {
		return "", "", err
	}

	outbuf := &bytes.Buffer{}
	if options.Stdout == nil {
		options.Stdout = outbuf
	}

	errbuf := &bytes.Buffer{}
	if options.Stderr == nil {
		options.Stderr = errbuf
	}

	errCh := execStream(ctx, kubeCli, config, options)
	err = <-errCh
	return strings.TrimSpace(outbuf.String()), strings.TrimSpace(errbuf.String()), errors.Wrap(err, "Failed to exec command in pod")
}

func execStream(ctx context.Context, kubeCli kubernetes.Interface, config *restclient.Config, options ExecOptions) chan error {
	const tty = false
	req := kubeCli.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(options.PodName).
		Namespace(options.Namespace).
		SubResource("exec")

	if len(options.ContainerName) != 0 {
		req.Param("container", options.ContainerName)
	}

	req.VersionedParams(&v1.PodExecOptions{
		Container: options.ContainerName,
		Command:   options.Command,
		Stdin:     options.Stdin != nil,
		Stdout:    options.Stdout != nil,
		Stderr:    options.Stderr != nil,
		TTY:       tty,
	}, scheme.ParameterCodec)

	errCh := make(chan error, 1)
	go func() {
		err := execute(
			ctx,
			"POST",
			req.URL(),
			config,
			options.Stdin,
			options.Stdout,
			options.Stderr,
			tty)
		errCh <- err
	}()

	return errCh
}

func execute(ctx context.Context, method string, url *url.URL, config *restclient.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool) error {
	exec, err := remotecommand.NewSPDYExecutor(config, method, url)
	if err != nil {
		return err
	}

	return exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    tty,
	})
}
