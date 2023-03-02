package kanctl

import (
	"context"
	"os"

	"github.com/kanisterio/kanister/pkg/secrets/repositoryserver"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sYAML "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
)

func performRepoServerSecretsValidation(p *validateParams) error {
	ctx := context.Background()
	cli, _, _, err := initializeClients()
	if err != nil {
		return errors.Wrap(err, "could not initialize clients for validation")
	}
	secret, err := getSecretFromCmd(ctx, cli, p)
	if err != nil {
		return err
	}
	return repositoryserver.ValidateSecret(secret)
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
		defer f.Close()
	}
	d := k8sYAML.NewYAMLOrJSONDecoder(f, 4096)
	secret := &corev1.Secret{}
	err = d.Decode(secret)
	if err != nil {
		return nil, err
	}
	return secret, nil
}
