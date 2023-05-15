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
	tlsSecretFlag                  = "tls"
	repoServerUserAccessSecretFlag = "repoServerUserAccessSecret"
	repoAdminUserSecretFlag        = "repoAdminUserSecret"
	repoPasswordSecretFlag         = "repoPasswordSecret"
	repoServerUserFlag             = "repoServerUser"
	repoServerAdminUserFlag        = "repoServerAdminUser"
	s3LocationCredsSecretFlag      = "s3LocationCredsSecret"
	s3LocationSecretFlag           = "s3LocationSecret"
	defaultKanisterNamespace       = "kanister"
	defaultRepositoryServerHost    = "localhost"
)

type repositoryServerParams struct {
	tls                    string
	tlsNS                  string
	repoServerUserAccess   string
	repoServerUserAccessNS string
	repoAdminUser          string
	repoAdminUserNS        string
	repoPassword           string
	repoPasswordNS         string
	repoServerUser         string
	repoServerAdminUser    string
	s3Location             string
	s3LocationNS           string
	s3LocationCreds        string
	s3LocationCredsNS      string
	prefix                 string
}

func newRepositoryServerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repository-server",
		Short: "Create a new RepositoryServer",
		Args:  cobra.ExactArgs(0),
	}

	cmd.AddCommand(newS3CompliantRepositoryServerCmd())
	cmd.PersistentFlags().StringP(tlsSecretFlag, "t", "", "name of the tls secret needed for secure client server communication")
	cmd.PersistentFlags().StringP(repoServerUserFlag, "u", "", "name of the user to be created for the kopia repository server")
	cmd.PersistentFlags().StringP(repoServerUserAccessSecretFlag, "k", "", "name of the secret having user access password and host")
	cmd.PersistentFlags().StringP(repoAdminUserSecretFlag, "a", "", "name of the secret for the repository server admin user details")
	cmd.PersistentFlags().StringP(repoPasswordSecretFlag, "r", "", "name of the secret containing password for the kopia repository")
	cmd.PersistentFlags().StringP(prefixFlag, "p", "", "prefix to be set in kopia repository")
	cmd.PersistentFlags().StringP(repoServerAdminUserFlag, "z", "", "kopia repository server admin user name")

	_ = cmd.MarkFlagRequired(tlsSecretFlag)
	_ = cmd.MarkFlagRequired(repoServerUserFlag)
	_ = cmd.MarkFlagRequired(repoServerUserAccessSecretFlag)
	_ = cmd.MarkFlagRequired(repoAdminUserSecretFlag)
	_ = cmd.MarkFlagRequired(repoPasswordSecretFlag)
	_ = cmd.MarkFlagRequired(prefixFlag)
	return cmd
}

func newS3CompliantRepositoryServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "s3compliant",
		Short: "Create a new kopia repository server for s3 compliant storage",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return createNewRepositoryServer(cmd, args)
		},
	}

	cmd.PersistentFlags().StringP(s3LocationSecretFlag, "l", "", "name of the secret containing the s3 location details")
	cmd.PersistentFlags().StringP(s3LocationCredsSecretFlag, "c", "", "name of the secret containing the credentials for s3")

	_ = cmd.MarkFlagRequired(s3LocationSecretFlag)
	_ = cmd.MarkFlagRequired(s3LocationCredsSecretFlag)
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

	repositoryServer, err := validateAndConstructRepositoryServer(rsParams)
	if err != nil {
		return err
	}
	_, crCli, _, _ := initializeClients()
	ctx := context.Background()
	rs, err := crCli.CrV1alpha1().RepositoryServers(defaultKanisterNamespace).Create(ctx, repositoryServer, metav1.CreateOptions{})
	if err != nil {
		return err
	} else {
		fmt.Printf("repository-server '%s' created\n", rs.GetName())
	}
	return nil
}

func generateRepositoryServerParams(cmd *cobra.Command) (*repositoryServerParams, error) {
	// Fetch values of the flags
	tls, _ := cmd.Flags().GetString(tlsSecretFlag)
	repoServerUser, _ := cmd.Flags().GetString(repoServerUserFlag)
	repoServerUserAccess, _ := cmd.Flags().GetString(repoServerUserAccessSecretFlag)
	repoAdminUser, _ := cmd.Flags().GetString(repoAdminUserSecretFlag)
	repoPassword, _ := cmd.Flags().GetString(repoPasswordSecretFlag)
	s3Location, _ := cmd.Flags().GetString(s3LocationSecretFlag)
	s3LocationCreds, _ := cmd.Flags().GetString(s3LocationCredsSecretFlag)
	prefix, _ := cmd.Flags().GetString(prefixFlag)
	repoServerAdminUser, _ := cmd.Flags().GetString(repoServerAdminUserFlag)

	var tlsNS, repoServerUserAccessNS, repoAdminUserNS, repoPasswordNS, s3LocationNS, s3LocationCredsNS string
	if strings.Contains(tls, "/") {
		fullTLS := strings.Split(tls, "/")
		if len(fullTLS) != 2 {
			return nil, errors.Errorf("Invalid secret name %s, it should be of the form namespace/name )", tls)
		}
		tlsNS = fullTLS[0]
		tls = fullTLS[1]
	} else {
		tlsNS = defaultKanisterNamespace
	}
	if strings.Contains(repoServerUserAccess, "/") {
		fullRepoServerUserAccess := strings.Split(repoServerUserAccess, "/")
		if len(fullRepoServerUserAccess) != 2 {
			return nil, errors.Errorf("Invalid secret name %s, it should be of the form namespace/name )", repoServerUserAccess)
		}
		repoServerUserAccessNS = fullRepoServerUserAccess[0]
		repoServerUserAccess = fullRepoServerUserAccess[1]
	} else {
		repoServerUserAccessNS = defaultKanisterNamespace
	}
	if strings.Contains(repoAdminUser, "/") {
		fullRepoAdminUser := strings.Split(repoAdminUser, "/")
		if len(fullRepoAdminUser) != 2 {
			return nil, errors.Errorf("Invalid secret name %s, it should be of the form namespace/name )", repoAdminUser)
		}
		repoAdminUserNS = fullRepoAdminUser[0]
		repoAdminUser = fullRepoAdminUser[1]
	} else {
		repoAdminUserNS = defaultKanisterNamespace
	}
	if strings.Contains(repoPassword, "/") {
		fullRepoPassword := strings.Split(repoPassword, "/")
		if len(fullRepoPassword) != 2 {
			return nil, errors.Errorf("Invalid secret name %s, it should be of the form namespace/name )", repoPassword)
		}
		repoPasswordNS = fullRepoPassword[0]
		repoPassword = fullRepoPassword[1]
	} else {
		repoPasswordNS = defaultKanisterNamespace
	}
	if strings.Contains(s3Location, "/") {
		fullS3Location := strings.Split(s3Location, "/")
		if len(fullS3Location) != 2 {
			return nil, errors.Errorf("Invalid secret name %s, it should be of the form namespace/name )", s3Location)
		}
		s3LocationNS = fullS3Location[0]
		s3Location = fullS3Location[1]
	} else {
		s3LocationNS = defaultKanisterNamespace
	}
	if strings.Contains(s3LocationCreds, "/") {
		fullS3LocationCreds := strings.Split(s3LocationCreds, "/")
		if len(fullS3LocationCreds) != 2 {
			return nil, errors.Errorf("Invalid secret name %s, it should be of the form namespace/name )", s3LocationCreds)
		}
		s3LocationCredsNS = fullS3LocationCreds[0]
		s3LocationCreds = fullS3LocationCreds[1]
	} else {
		s3LocationCredsNS = defaultKanisterNamespace
	}

	return &repositoryServerParams{
		tls:                    tls,
		tlsNS:                  tlsNS,
		repoServerUserAccess:   repoServerUserAccess,
		repoServerUserAccessNS: repoServerUserAccessNS,
		repoAdminUser:          repoAdminUser,
		repoAdminUserNS:        repoAdminUserNS,
		repoPassword:           repoPassword,
		repoPasswordNS:         repoPasswordNS,
		repoServerUser:         repoServerUser,
		repoServerAdminUser:    repoServerAdminUser,
		s3Location:             s3Location,
		s3LocationNS:           s3LocationNS,
		s3LocationCreds:        s3LocationCreds,
		s3LocationCredsNS:      s3LocationCredsNS,
		prefix:                 prefix,
	}, nil
}

func validateAndConstructRepositoryServer(rsParams *repositoryServerParams) (*v1alpha1.RepositoryServer, error) {
	// Fetch and Validate Secrets
	ctx := context.Background()
	cli, _, _, err := initializeClients()
	if err != nil {
		return nil, err
	}
	tlsSecret, err := cli.CoreV1().Secrets(rsParams.tlsNS).Get(ctx, rsParams.tls, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	repoServerUserAccessSecret, err := cli.CoreV1().Secrets(rsParams.repoServerUserAccessNS).Get(ctx, rsParams.repoServerUserAccess, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	repoAdminUserSecret, err := cli.CoreV1().Secrets(rsParams.repoAdminUserNS).Get(ctx, rsParams.repoAdminUser, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	repoPasswordSecret, err := cli.CoreV1().Secrets(rsParams.repoPasswordNS).Get(ctx, rsParams.repoPassword, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	s3LocationSecret, err := cli.CoreV1().Secrets(rsParams.s3LocationNS).Get(ctx, rsParams.s3Location, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	s3LocationCredsSecret, err := cli.CoreV1().Secrets(rsParams.s3LocationCredsNS).Get(ctx, rsParams.s3LocationCreds, metav1.GetOptions{})
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
					Name:      s3LocationSecret.GetName(),
					Namespace: s3LocationSecret.GetNamespace(),
				},
				CredentialSecretRef: corev1.SecretReference{
					Name:      s3LocationCredsSecret.GetName(),
					Namespace: s3LocationCredsSecret.GetNamespace(),
				},
			},
			Repository: v1alpha1.Repository{
				RootPath: rsParams.prefix,
				Username: rsParams.repoServerAdminUser,
				Hostname: defaultRepositoryServerHost,
				PasswordSecretRef: corev1.SecretReference{
					Name:      repoPasswordSecret.GetName(),
					Namespace: repoPasswordSecret.GetNamespace(),
				},
			},
			Server: v1alpha1.Server{
				UserAccess: v1alpha1.UserAccess{
					UserAccessSecretRef: corev1.SecretReference{
						Name:      repoServerUserAccessSecret.GetName(),
						Namespace: repoServerUserAccessSecret.GetNamespace(),
					},
					Username: rsParams.repoServerUser,
				},
				AdminSecretRef: corev1.SecretReference{
					Name:      repoAdminUserSecret.GetName(),
					Namespace: repoAdminUserSecret.GetNamespace(),
				},
				TLSSecretRef: corev1.SecretReference{
					Name:      tlsSecret.GetName(),
					Namespace: tlsSecret.GetNamespace(),
				},
			},
		},
	}, nil
}
