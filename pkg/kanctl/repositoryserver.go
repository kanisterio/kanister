// Copyright 2023 The Kanister Authors.
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

package kanctl

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
)

const (
	tlsSecretFlag                  = "tlsSecret"
	repoServerUserAccessSecretFlag = "repositoryServerUserAccessSecret"
	repoAdminUserSecretFlag        = "repositoryAdminUserSecret"
	repoPasswordSecretFlag         = "repositoryPasswordSecret"
	repoServerUserFlag             = "repositoryServerUser"
	repoServerAdminUserFlag        = "repositoryServerAdminUser"
	locationCredsSecretFlag        = "locationCredsSecret"
	locationSecretFlag             = "locationSecret"
	defaultKanisterNamespace       = "kanister"
	defaultRepositoryServerHost    = "localhost"
)

type repositoryServerParams struct {
	tls                             string
	repositoryServerUser            string
	repositoryServerUserAccess      string
	repositoryServerAdminUser       string
	repositoryServerAdminUserAccess string
	repositoryPassword              string
	prefix                          string
	location                        string
	locationCreds                   string
}

func newRepositoryServerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repository-server",
		Short: "Create a new RepositoryServer",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return createNewRepositoryServer(cmd, args)
		},
	}

	cmd.PersistentFlags().StringP(tlsSecretFlag, "t", "", "name of the tls secret needed for secure kopia client and kopia repository server communication")
	cmd.PersistentFlags().StringP(repoServerUserFlag, "u", "", "name of the user to be created for the kopia repository server")
	cmd.PersistentFlags().StringP(repoServerUserAccessSecretFlag, "k", "", "name of the secret having access credentials of the users that can connect to kopia repository server")
	cmd.PersistentFlags().StringP(repoAdminUserSecretFlag, "a", "", "name of the secret having admin credentials to connect to connect to kopia repository server")
	cmd.PersistentFlags().StringP(repoPasswordSecretFlag, "r", "", "name of the secret containing password for the kopia repository")
	cmd.PersistentFlags().StringP(prefixFlag, "p", "", "prefix to be set in kopia repository")
	cmd.PersistentFlags().StringP(repoServerAdminUserFlag, "z", "", "kopia repository server admin user name")
	cmd.PersistentFlags().StringP(locationSecretFlag, "l", "", "name of the secret containing kopia repository storage location details")
	cmd.PersistentFlags().StringP(locationCredsSecretFlag, "c", "", "name of the secret containing kopia repository storage credentials")

	_ = cmd.MarkFlagRequired(tlsSecretFlag)
	_ = cmd.MarkFlagRequired(repoServerUserFlag)
	_ = cmd.MarkFlagRequired(repoServerUserAccessSecretFlag)
	_ = cmd.MarkFlagRequired(repoAdminUserSecretFlag)
	_ = cmd.MarkFlagRequired(repoPasswordSecretFlag)
	_ = cmd.MarkFlagRequired(prefixFlag)
	_ = cmd.MarkFlagRequired(locationSecretFlag)
	_ = cmd.MarkFlagRequired(locationCredsSecretFlag)
	return cmd
}

func createNewRepositoryServer(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	if len(args) > 0 {
		return newArgsLengthError("expected 0 args. Got %#v", args)
	}

	rsParams, err := generateRepositoryServerParams(cmd)
	if err != nil {
		return err
	}

	repositoryServer, err := validateSecretsAndConstructRepositoryServer(rsParams)
	if err != nil {
		return err
	}
	_, crCli, _, _ := initializeClients()
	ctx := context.Background()
	rs, err := crCli.CrV1alpha1().RepositoryServers(defaultKanisterNamespace).Create(ctx, repositoryServer, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	fmt.Printf("repository-server '%s' created\n", rs.GetName())
	return nil
}

func generateRepositoryServerParams(cmd *cobra.Command) (*repositoryServerParams, error) {
	// Fetch values of the flags
	tlsSecret, _ := cmd.Flags().GetString(tlsSecretFlag)
	repositoryServerUser, _ := cmd.Flags().GetString(repoServerUserFlag)
	repositoryServerUserAccessSecret, _ := cmd.Flags().GetString(repoServerUserAccessSecretFlag)
	repositoryServerAdminUser, _ := cmd.Flags().GetString(repoServerAdminUserFlag)
	repositoryServerAdminUserAccessSecret, _ := cmd.Flags().GetString(repoAdminUserSecretFlag)
	repositoryPassword, _ := cmd.Flags().GetString(repoPasswordSecretFlag)
	prefix, _ := cmd.Flags().GetString(prefixFlag)
	location, _ := cmd.Flags().GetString(locationSecretFlag)
	locationCreds, _ := cmd.Flags().GetString(locationCredsSecretFlag)

	if strings.Contains(tlsSecret, "/") {
		return nil, errors.Errorf("Invalid secret name %s, it should not be of the form namespace/name )", tlsSecret)
	}
	if strings.Contains(repositoryServerUserAccessSecret, "/") {
		return nil, errors.Errorf("Invalid secret name %s, it should not be of the form namespace/name )", repositoryServerUserAccessSecret)
	}
	if strings.Contains(repositoryServerAdminUserAccessSecret, "/") {
		return nil, errors.Errorf("Invalid secret name %s, it should not be of the form namespace/name )", repositoryServerAdminUserAccessSecret)
	}
	if strings.Contains(repositoryPassword, "/") {
		return nil, errors.Errorf("Invalid secret name %s, it should not be of the form namespace/name )", repositoryPassword)
	}
	if strings.Contains(location, "/") {
		return nil, errors.Errorf("Invalid secret name %s, it should not be of the form namespace/name )", location)
	}
	if strings.Contains(locationCreds, "/") {
		return nil, errors.Errorf("Invalid secret name %s, it should not be of the form namespace/name )", locationCreds)
	}

	return &repositoryServerParams{
		tls:                             tlsSecret,
		repositoryServerUser:            repositoryServerUser,
		repositoryServerUserAccess:      repositoryServerUserAccessSecret,
		repositoryServerAdminUser:       repositoryServerAdminUser,
		repositoryServerAdminUserAccess: repositoryServerAdminUserAccessSecret,
		repositoryPassword:              repositoryPassword,
		prefix:                          prefix,
		location:                        location,
		locationCreds:                   locationCreds,
	}, nil
}

func validateSecretsAndConstructRepositoryServer(rsParams *repositoryServerParams) (*v1alpha1.RepositoryServer, error) {
	// Fetch and Validate Secrets
	ctx := context.Background()
	cli, _, _, err := initializeClients()
	if err != nil {
		return nil, err
	}
	tlsSecret, err := cli.CoreV1().Secrets(defaultKanisterNamespace).Get(ctx, rsParams.tls, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	repositoryServerUserAccessSecret, err := cli.CoreV1().Secrets(defaultKanisterNamespace).Get(ctx, rsParams.repositoryServerUserAccess, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	repositoryServerAdminUserAccessSecret, err := cli.CoreV1().Secrets(defaultKanisterNamespace).Get(ctx, rsParams.repositoryServerAdminUserAccess, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	repositoryPasswordSecret, err := cli.CoreV1().Secrets(defaultKanisterNamespace).Get(ctx, rsParams.repositoryPassword, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	locationSecret, err := cli.CoreV1().Secrets(defaultKanisterNamespace).Get(ctx, rsParams.location, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	locationCredsSecret, err := cli.CoreV1().Secrets(defaultKanisterNamespace).Get(ctx, rsParams.locationCreds, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return &v1alpha1.RepositoryServer{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: `kopia-repo-server-`,
		},
		Spec: v1alpha1.RepositoryServerSpec{
			Storage: v1alpha1.Storage{
				SecretRef: corev1.SecretReference{
					Name:      locationSecret.GetName(),
					Namespace: locationSecret.GetNamespace(),
				},
				CredentialSecretRef: corev1.SecretReference{
					Name:      locationCredsSecret.GetName(),
					Namespace: locationCredsSecret.GetNamespace(),
				},
			},
			Repository: v1alpha1.Repository{
				RootPath: rsParams.prefix,
				Username: rsParams.repositoryServerAdminUser,
				Hostname: defaultRepositoryServerHost,
				PasswordSecretRef: corev1.SecretReference{
					Name:      repositoryPasswordSecret.GetName(),
					Namespace: repositoryPasswordSecret.GetNamespace(),
				},
			},
			Server: v1alpha1.Server{
				UserAccess: v1alpha1.UserAccess{
					UserAccessSecretRef: corev1.SecretReference{
						Name:      repositoryServerUserAccessSecret.GetName(),
						Namespace: repositoryServerUserAccessSecret.GetNamespace(),
					},
					Username: rsParams.repositoryServerUser,
				},
				AdminSecretRef: corev1.SecretReference{
					Name:      repositoryServerAdminUserAccessSecret.GetName(),
					Namespace: repositoryServerAdminUserAccessSecret.GetNamespace(),
				},
				TLSSecretRef: corev1.SecretReference{
					Name:      tlsSecret.GetName(),
					Namespace: tlsSecret.GetNamespace(),
				},
			},
		},
	}, nil
}
