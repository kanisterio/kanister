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
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/poll"
)

const (
	tlsSecretFlag                       = "tls-secret"
	repoServerUserFlag                  = "user"
	repoServerUserAccessSecretFlag      = "user-access-secret"
	repoServerAdminUserAccessSecretFlag = "admin-user-access-secret"
	kopiaRepoPasswordSecretFlag         = "kopia-repository-password-secret"
	kopiaRepoUserFlag                   = "kopia-repository-user"
	locationCredsSecretFlag             = "location-creds-secret"
	locationSecretFlag                  = "location-secret"
	defaultRepositoryServerHost         = "localhost"
	waitFlag                            = "wait"
	contextWaitTimeout                  = 10 * time.Minute
)

type repositoryServerParams struct {
	tls                             string
	repositoryServerUser            string
	repositoryServerUserAccess      string
	repositoryServerAdminUserAccess string
	repositoryUser                  string
	repositoryPassword              string
	prefix                          string
	location                        string
	locationCreds                   string
	namespace                       string
}

func newRepositoryServerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repository-server",
		Short: "Create a new RepositoryServer",
		Args:  cobra.ExactArgs(0),
		RunE:  createNewRepositoryServer,
	}

	cmd.PersistentFlags().StringP(tlsSecretFlag, "t", "", "name of the tls secret needed for secure kopia client and kopia repository server communication")
	cmd.PersistentFlags().StringP(repoServerUserFlag, "u", "", "name of the user to be created for the kopia repository server")
	cmd.PersistentFlags().StringP(repoServerUserAccessSecretFlag, "s", "", "name of the secret having access credentials of the users that can connect to kopia repository server")
	cmd.PersistentFlags().StringP(repoServerAdminUserAccessSecretFlag, "a", "", "name of the secret having admin credentials to connect to connect to kopia repository server")
	cmd.PersistentFlags().StringP(kopiaRepoPasswordSecretFlag, "r", "", "name of the secret containing password for the kopia repository")
	cmd.PersistentFlags().StringP(prefixFlag, "p", "", "prefix to be set in kopia repository")
	cmd.PersistentFlags().StringP(kopiaRepoUserFlag, "k", "", "name of the user for accessing the kopia repository")
	cmd.PersistentFlags().StringP(locationSecretFlag, "l", "", "name of the secret containing kopia repository storage location details")
	cmd.PersistentFlags().StringP(locationCredsSecretFlag, "c", "", "name of the secret containing kopia repository storage credentials")
	cmd.PersistentFlags().BoolP(waitFlag, "w", false, "wait for the kopia repository server to be in ready state after creation")

	_ = cmd.MarkFlagRequired(tlsSecretFlag)
	_ = cmd.MarkFlagRequired(repoServerUserFlag)
	_ = cmd.MarkFlagRequired(repoServerUserAccessSecretFlag)
	_ = cmd.MarkFlagRequired(repoServerAdminUserAccessSecretFlag)
	_ = cmd.MarkFlagRequired(kopiaRepoPasswordSecretFlag)
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

	config, err := kube.LoadConfig()
	if err != nil {
		return err
	}
	cli, err := kubernetes.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "could not get the kubernetes client")
	}
	crCli, err := versioned.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "could not get the CRD client")
	}

	ctx := context.Background()
	rs, err := crCli.CrV1alpha1().RepositoryServers(rsParams.namespace).Create(ctx, repositoryServer, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	fmt.Printf("repositoryservers.cr.kanister.io/%s created\n", rs.GetName())

	waitFlag, _ := cmd.Flags().GetBool(waitFlag)
	if waitFlag {
		fmt.Print("Waiting for the kopia repository server CR to be in ready state...\n")
		err = waitForRepositoryServerReady(ctx, cli, crCli, rs)
		if err != nil {
			return err
		}
		fmt.Printf("repositoryservers.cr.kanister.io/%s is ready.\n", rs.GetName())
	}
	return nil
}

func generateRepositoryServerParams(cmd *cobra.Command) (*repositoryServerParams, error) {
	// Fetch values of the flags
	tlsSecret, _ := cmd.Flags().GetString(tlsSecretFlag)
	if strings.Contains(tlsSecret, "/") {
		return nil, errors.Errorf("Invalid secret name %s, it should not be of the form namespace/name )", tlsSecret)
	}

	repositoryServerUser, _ := cmd.Flags().GetString(repoServerUserFlag)

	repositoryServerUserAccessSecret, _ := cmd.Flags().GetString(repoServerUserAccessSecretFlag)
	if strings.Contains(repositoryServerUserAccessSecret, "/") {
		return nil, errors.Errorf("Invalid secret name %s, it should not be of the form namespace/name )", repositoryServerUserAccessSecret)
	}

	repositoryServerAdminUserAccessSecret, _ := cmd.Flags().GetString(repoServerAdminUserAccessSecretFlag)
	if strings.Contains(repositoryServerAdminUserAccessSecret, "/") {
		return nil, errors.Errorf("Invalid secret name %s, it should not be of the form namespace/name )", repositoryServerAdminUserAccessSecret)
	}

	repositoryUser, _ := cmd.Flags().GetString(kopiaRepoUserFlag)

	repositoryPassword, _ := cmd.Flags().GetString(kopiaRepoPasswordSecretFlag)
	if strings.Contains(repositoryPassword, "/") {
		return nil, errors.Errorf("Invalid secret name %s, it should not be of the form namespace/name )", repositoryPassword)
	}

	prefix, _ := cmd.Flags().GetString(prefixFlag)

	location, _ := cmd.Flags().GetString(locationSecretFlag)
	if strings.Contains(location, "/") {
		return nil, errors.Errorf("Invalid secret name %s, it should not be of the form namespace/name )", location)
	}

	locationCreds, _ := cmd.Flags().GetString(locationCredsSecretFlag)
	if strings.Contains(locationCreds, "/") {
		return nil, errors.Errorf("Invalid secret name %s, it should not be of the form namespace/name )", locationCreds)
	}

	ns, err := resolveNamespace(cmd)
	if err != nil {
		return nil, err
	}

	return &repositoryServerParams{
		tls:                             tlsSecret,
		repositoryServerUser:            repositoryServerUser,
		repositoryServerUserAccess:      repositoryServerUserAccessSecret,
		repositoryUser:                  repositoryUser,
		repositoryServerAdminUserAccess: repositoryServerAdminUserAccessSecret,
		repositoryPassword:              repositoryPassword,
		prefix:                          prefix,
		location:                        location,
		locationCreds:                   locationCreds,
		namespace:                       ns,
	}, nil
}

func validateSecretsAndConstructRepositoryServer(rsParams *repositoryServerParams) (*crv1alpha1.RepositoryServer, error) {
	// Fetch and Validate Secrets
	ctx := context.Background()
	config, err := kube.LoadConfig()
	if err != nil {
		return nil, err
	}
	cli, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "could not get the kubernetes client")
	}
	tlsSecret, err := cli.CoreV1().Secrets(rsParams.namespace).Get(ctx, rsParams.tls, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	repositoryServerUserAccessSecret, err := cli.CoreV1().Secrets(rsParams.namespace).Get(ctx, rsParams.repositoryServerUserAccess, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	repositoryServerAdminUserAccessSecret, err := cli.CoreV1().Secrets(rsParams.namespace).Get(ctx, rsParams.repositoryServerAdminUserAccess, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	repositoryPasswordSecret, err := cli.CoreV1().Secrets(rsParams.namespace).Get(ctx, rsParams.repositoryPassword, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	locationSecret, err := cli.CoreV1().Secrets(rsParams.namespace).Get(ctx, rsParams.location, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	locationCredsSecret, err := cli.CoreV1().Secrets(rsParams.namespace).Get(ctx, rsParams.locationCreds, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return &crv1alpha1.RepositoryServer{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: `kopia-repo-server-`,
		},
		Spec: crv1alpha1.RepositoryServerSpec{
			Storage: crv1alpha1.Storage{
				SecretRef: corev1.SecretReference{
					Name:      locationSecret.GetName(),
					Namespace: locationSecret.GetNamespace(),
				},
				CredentialSecretRef: corev1.SecretReference{
					Name:      locationCredsSecret.GetName(),
					Namespace: locationCredsSecret.GetNamespace(),
				},
			},
			Repository: crv1alpha1.Repository{
				RootPath: rsParams.prefix,
				Username: rsParams.repositoryUser,
				Hostname: defaultRepositoryServerHost,
				PasswordSecretRef: corev1.SecretReference{
					Name:      repositoryPasswordSecret.GetName(),
					Namespace: repositoryPasswordSecret.GetNamespace(),
				},
			},
			Server: crv1alpha1.Server{
				UserAccess: crv1alpha1.UserAccess{
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

func waitForRepositoryServerReady(ctx context.Context, cli *kubernetes.Clientset, crCli *versioned.Clientset, rs *crv1alpha1.RepositoryServer) error {
	timeoutCtx, waitCancel := context.WithTimeout(ctx, contextWaitTimeout)
	defer waitCancel()
	pollErr := poll.Wait(timeoutCtx, func(ctx context.Context) (bool, error) {
		repositoryServer, err := crCli.CrV1alpha1().RepositoryServers(rs.GetNamespace()).Get(ctx, rs.GetName(), metav1.GetOptions{})
		if repositoryServer.Status.Progress == crv1alpha1.Ready && err == nil {
			return true, nil
		}
		return false, err
	})
	if pollErr != nil {
		repositoryServer, err := crCli.CrV1alpha1().RepositoryServers(rs.GetNamespace()).Get(ctx, rs.GetName(), metav1.GetOptions{})
		if err != nil {
			return errors.Wrapf(err, "Error Getting repository server %s", repositoryServer.GetName())
		}

		opts := metav1.ListOptions{
			FieldSelector: fmt.Sprintf("involvedObject.name=%s", repositoryServer.GetName()),
		}
		events, err := cli.CoreV1().Events(repositoryServer.GetNamespace()).List(ctx, opts)
		if err != nil {
			return err
		}

		return errors.Wrapf(pollErr, "Repository Server is not ready.\nCurrent Status: %s\nReason: %s\n", repositoryServer.Status.Progress, events.Items[0].Message)
	}
	return nil
}
