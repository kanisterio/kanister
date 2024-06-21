// Copyright 2023 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kanctl

import (
	"context"
	"os"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sYAML "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/secrets"
)

func performRepoServerSecretsValidation(ctx context.Context, p *validateParams) error {
	var cli kubernetes.Interface
	var secret *corev1.Secret

	cli, err := kube.NewClient()
	if err != nil {
		return errors.Wrap(err, "could not get the kubernetes client")
	}

	secret, err = getSecretFromCmd(ctx, cli, p)
	if err != nil {
		return err
	}
	return secrets.ValidateRepositoryServerSecret(secret)
}

func getSecretFromCmd(ctx context.Context, cli kubernetes.Interface, p *validateParams) (*corev1.Secret, error) {
	if p.name != "" {
		return cli.CoreV1().Secrets(p.namespace).Get(ctx, p.name, metav1.GetOptions{})
	}
	return getSecretFromFile(ctx, p.filename)
}

func getSecretFromFile(ctx context.Context, filename string) (*corev1.Secret, error) {
	var f *os.File
	var err error

	if filename == "-" {
		f = os.Stdin
	} else {
		f, err = os.Open(filename)
		if err != nil {
			return nil, err
		}
		defer f.Close() //nolint:errcheck
	}
	d := k8sYAML.NewYAMLOrJSONDecoder(f, 4096)
	secret := &corev1.Secret{}
	err = d.Decode(secret)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode the secret passed")
	}
	return secret, nil
}
