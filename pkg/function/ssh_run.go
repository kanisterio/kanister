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

package function

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"strings"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	SSHRunFuncName = "SSHRun"
	// The instance you are going to run the commands into
	SSHRunInstanceIPArg = "instanceIP"
	// The user, you will be logging into the above instance with
	// Like in example `ssh user@instanceIP`
	SSHRunUserArg = "user"
	// Authentication mechanism that is going to be used to connect to above instance
	SSHRunAuthMechArg = "authMechanism"
	// Secret that is going to hold details, acc to auth mech mentioned above
	// For ex if auth mech is password, the password is going to be in this secret
	SSHRunSecretNameArg = "secret"
	//Namespace the above secret is present in
	SSHRunNamespaceArg = "namespace"
	// Command that is to be run on the instance
	SSHRunCommandArg = "command"

	SSHPort = "22"

	// These are some of the auth mechanisms that we support as of now
	PwdAuthMec           authMech = "password"
	KeyAuthMech          authMech = "SSHKey"
	ProtectedKeyAuthMech authMech = "protectedSSHKey"
)

type authMech string

type sshRunFunc struct{}

func init() {
	_ = kanister.Register(&sshRunFunc{})
}

var (
	_ kanister.Func = (*sshRunFunc)(nil)
)

func (*sshRunFunc) Name() string {
	return SSHRunFuncName
}

func (f *sshRunFunc) RequiredArgs() []string {
	return []string{SSHRunInstanceIPArg, SSHRunUserArg, SSHRunAuthMechArg, SSHRunSecretNameArg, SSHRunNamespaceArg, SSHRunCommandArg}
}

func (f *sshRunFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	cli, err := kube.NewClient()
	if err != nil {
		return nil, err
	}

	var instanceIP, user, authMech, secretName, namespace string
	var cmd string

	if err := Arg(args, SSHRunInstanceIPArg, &instanceIP); err != nil {
		return nil, err
	}

	if err := Arg(args, SSHRunUserArg, &user); err != nil {
		return nil, err
	}

	if err := Arg(args, SSHRunAuthMechArg, &authMech); err != nil {
		return nil, err
	}

	if err := Arg(args, SSHRunSecretNameArg, &secretName); err != nil {
		return nil, err
	}

	if err := Arg(args, SSHRunNamespaceArg, &namespace); err != nil {
		return nil, err
	}

	if err := Arg(args, SSHRunCommandArg, &cmd); err != nil {
		return nil, err
	}

	authDetails, err := validateSecret(cli, authMech, secretName, namespace)
	if err != nil {
		return nil, err
	}

	if err := runCMD(authDetails, instanceIP, user, cmd, authMech); err != nil {
		return nil, errors.Wrapf(err, "Error running the command on provided instnace %s", instanceIP)
	}

	return nil, nil
}

func runCMD(authDetails interface{}, instanceIP, user string, cmd, authMech string) error {
	var config *ssh.ClientConfig
	var err error

	switch authMech {
	case string(PwdAuthMec):
		config, err = getPWDClientConfig(authDetails, user)
		if err != nil {
			return errors.Wrapf(err, "Error getting client config for the auth mechanism %s", authMech)
		}
	case string(KeyAuthMech):
		config, err = getKeyClientConfig(authDetails, user)
		if err != nil {
			return errors.Wrapf(err, "Error getting client config for the auth mechanism %s", authMech)
		}
	case string(ProtectedKeyAuthMech):
		config, err = getProtectedKeyClientConfig(authDetails, user)
		if err != nil {
			return errors.Wrapf(err, "Error getting client config for the auth mechanism %s", authMech)
		}
	}

	hostAddress := strings.Join([]string{instanceIP, SSHPort}, ":")
	client, err := ssh.Dial("tcp", hostAddress, config)
	if err != nil {
		return errors.Wrapf(err, "Error dialing connection to the instance %s", instanceIP)
	}

	session, err := client.NewSession()
	if err != nil {
		return errors.Wrapf(err, "Error getting tcp session to the instance %s", instanceIP)
	}

	var buffOut, buffErr bytes.Buffer
	session.Stdout = &buffOut
	session.Stderr = &buffErr

	if err := session.Run(cmd); err != nil {
		return errors.Wrapf(err, "Error running command on instance %s", instanceIP)
	}

	return nil
}

func getPWDClientConfig(authDetails interface{}, user string) (*ssh.ClientConfig, error) {
	password := authDetails.(string)

	return &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}, nil
}

func getKeyClientConfig(authDetails interface{}, user string) (*ssh.ClientConfig, error) {
	key, err := ssh.ParsePrivateKey(authDetails.([]byte))
	if err != nil {
		return nil, errors.Wrapf(err, "Error parsing private key")
	}

	return &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}, nil
}

func getProtectedKeyClientConfig(authDetails interface{}, user string) (*ssh.ClientConfig, error) {
	protectedKeyFields := authDetails.(protectedKeyFields)

	key, err := ssh.ParsePrivateKeyWithPassphrase(protectedKeyFields.privateKey, []byte(protectedKeyFields.passphrase))
	if err != nil {
		return nil, errors.Wrapf(err, "Error parsing private key with passphrase")
	}

	return &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}, nil
}

type protectedKeyFields struct {
	privateKey []byte
	passphrase string
}

// validateSecret validates if necessary fields are provided in the secret acc to authMech and returns those fields
func validateSecret(cli kubernetes.Interface, authMech, secretName, namespace string) (interface{}, error) {
	secret, err := cli.CoreV1().Secrets(namespace).Get(secretName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "Error getting secrect %s, in namespace %s", secretName, namespace)
	}

	switch authMech {
	case string(PwdAuthMec):
		password, ok := secret.Data["password"]
		if !ok {
			return nil, errors.New(fmt.Sprintf("Key '%s' is required in the secret for authentication mechanism %s", "password", authMech))
		}
		return string(password), nil
	case string(KeyAuthMech):
		privateKey, ok := secret.Data["id_rsa"]
		if !ok {
			return nil, errors.New(fmt.Sprintf("Key '%s' is required in the secret for authentication mechanism %s", "id_rsa", authMech))
		}
		return privateKey, nil
	case string(ProtectedKeyAuthMech):
		pvtKey, ok1 := secret.Data["id_rsa"]
		passPhrase, ok2 := secret.Data["passPhrase"]

		if !ok1 || !ok2 {
			return nil, errors.New(fmt.Sprintf("Keys '%s' and '%s' are required in the secret for authentication mechanism %s", "id_rsa", "passPhrase", authMech))
		}
		return protectedKeyFields{
			privateKey: pvtKey,
			passphrase: string(passPhrase),
		}, nil
	default:
		return nil, errors.New(fmt.Sprintf("Provided authentication (%s) mechanism is not supported", authMech))
	}
}
