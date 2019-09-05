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
	"io"
	"io/ioutil"
	"net/url"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
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

	Stdin         io.Reader
	CaptureStdout bool
	CaptureStderr bool
}

// Exec is our version of the call to `kubectl exec` that does not depend on
// k8s.io/kubernetes.
func Exec(cli kubernetes.Interface, namespace, pod, container string, command []string, stdin io.Reader) (string, string, error) {
	opts := ExecOptions{
		Command:       command,
		Namespace:     namespace,
		PodName:       pod,
		ContainerName: container,
		Stdin:         stdin,
		CaptureStdout: true,
		CaptureStderr: true,
	}
	return ExecWithOptions(cli, opts)
}

// ExecWithOptions executes a command in the specified container,
// returning stdout, stderr and error. `options` allowed for
// additional parameters to be passed.
func ExecWithOptions(kubeCli kubernetes.Interface, options ExecOptions) (string, string, error) {
	config, err := LoadConfig()
	if err != nil {
		return "", "", err
	}
	o, e, errCh := execStream(kubeCli, config, options)
	defer func() { _ = o.Close() }()
	defer func() { _ = e.Close() }()
	stdout, oerr := ioutil.ReadAll(o)
	if oerr != nil {
		log.Info("Failed to read stdout:", oerr.Error())
	}
	stderr, eerr := ioutil.ReadAll(e)
	if eerr != nil {
		log.Info("Failed to read stderr:", eerr.Error())
	}

	return string(stdout), string(stderr), <-errCh
}

func execStream(kubeCli kubernetes.Interface, config *restclient.Config, options ExecOptions) (io.ReadCloser, io.ReadCloser, chan error) {
	const tty = false
	req := kubeCli.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(options.PodName).
		Namespace(options.Namespace).
		SubResource("exec").
		Param("container", options.ContainerName)
	for _, c := range options.Command {
		req.Param("command", c)
	}
	req.VersionedParams(&v1.PodExecOptions{
		Container: options.ContainerName,
		Command:   options.Command,
		Stdin:     options.Stdin != nil,
		Stdout:    options.CaptureStdout,
		Stderr:    options.CaptureStderr,
		TTY:       tty,
	}, scheme.ParameterCodec)

	pro, pwo := io.Pipe()
	pre, pwe := io.Pipe()
	errCh := make(chan error, 1)
	go func() {
		err := execute("POST", req.URL(), config, options.Stdin, pwo, pwe, tty)
		errCh <- err
		_ = pwo.Close()
		_ = pwe.Close()
	}()
	return pro, pre, errCh
}

func execute(method string, url *url.URL, config *restclient.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool) error {
	exec, err := remotecommand.NewSPDYExecutor(config, method, url)
	if err != nil {
		return err
	}
	return exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    tty,
	})
}
