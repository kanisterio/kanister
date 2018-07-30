package kanctl

import (
	"context"
	"fmt"
	"os"
	"reflect"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sYAML "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned"
	"github.com/kanisterio/kanister/pkg/validate"
)

const (
	bucketFlag    = "bucket"
	endpointFlag  = "endpoint"
	prefixFlag    = "prefix"
	regionFlag    = "region"
	accessKeyFlag = "access-key"
	secretKeyFlag = "secret-key"

	idField           = "access_key_id"
	secretField       = "secret_access_key"
	skipSSLVerifyFlag = "skip-SSL-verification"

	schemaValidation      = "Validate Profile schema"
	regionValidation      = "Validate bucket region specified in profile"
	readAccessValidation  = "Validate read access to bucket specified in profile"
	writeAccessValidation = "Validate write access to bucket specified in profile"
)

type s3CompliantParams struct {
	namespace string
	bucket    string
	endpoint  string
	prefix    string
	region    string

	accessKey string
	secretKey string

	skipSSLVerify bool
}

func newProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Create a new profile",
		Args:  cobra.ExactArgs(0),
	}

	cmd.AddCommand(newS3CompliantProfileCmd())
	cmd.PersistentFlags().Bool(skipSSLVerifyFlag, false, "if set, SSL verification is disabled for the profile")
	return cmd
}

func newS3CompliantProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "s3compliant",
		Short: "Create new S3 compliant profile",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return createS3CompliantProfile(cmd, args)
		},
	}
	cmd.Flags().StringP(bucketFlag, "b", "", "s3 bucket name")
	cmd.Flags().StringP(endpointFlag, "e", "", "endpoint URL of the s3 bucket")
	cmd.Flags().StringP(prefixFlag, "p", "", "prefix URL of the s3 bucket")
	cmd.Flags().StringP(regionFlag, "r", "", "region of the s3 bucket")

	cmd.Flags().StringP(accessKeyFlag, "a", "", "access key of the s3 compliant bucket")
	cmd.Flags().StringP(secretKeyFlag, "s", "", "secret key of the s3 compliant bucket")

	cmd.MarkFlagRequired(bucketFlag)
	cmd.MarkFlagRequired(accessKeyFlag)
	cmd.MarkFlagRequired(secretKeyFlag)
	return cmd
}

func createS3CompliantProfile(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return newArgsLengthError("expected 0 args. Got %#v", args)
	}
	ctx := context.Background()
	skipValidation, _ := cmd.Flags().GetBool(skipValidationFlag)
	dryRun, _ := cmd.Flags().GetBool(dryRunFlag)
	cli, crCli, err := initializeClients()
	if err != nil {
		return err
	}
	s3P, err := getS3CompliantParams(cmd)
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true
	secret := constructSecret(s3P)
	profile := constructS3CompliantProfile(s3P, secret)
	if dryRun {
		// Just perform schema validation and print YAML
		if err := validate.ProfileSchema(profile); err != nil {
			return err
		}
		if err := printSecret(secret); err != nil {
			return err
		}
		fmt.Println("---")
		return printProfile(profile)
	}
	secret, err = createSecret(ctx, secret, cli)
	if err != nil {
		return errors.Wrap(err, "failed to create secret")
	}
	err = validateProfile(ctx, profile, cli, skipValidation, true)
	if err != nil {
		fmt.Printf("validation failed, deleting secret '%s'\n", secret.GetName())
		if rmErr := deleteSecret(ctx, secret, cli); rmErr != nil {
			return errors.Wrap(rmErr, "failed to delete secret after validation failed")
		}
		return errors.Wrap(err, "profile validation failed")
	}
	return createProfile(ctx, profile, crCli)
}

func getS3CompliantParams(cmd *cobra.Command) (*s3CompliantParams, error) {
	ns, err := resolveNamespace(cmd)
	if err != nil {
		return nil, err
	}
	// Location
	bucket, _ := cmd.Flags().GetString(bucketFlag)
	endpoint, _ := cmd.Flags().GetString(endpointFlag)
	prefix, _ := cmd.Flags().GetString(prefixFlag)
	region, _ := cmd.Flags().GetString(regionFlag)

	// Secret
	accessKey, _ := cmd.Flags().GetString(accessKeyFlag)
	secretKey, _ := cmd.Flags().GetString(secretKeyFlag)

	skipSSLVerify, _ := cmd.Flags().GetBool(skipSSLVerifyFlag)
	return &s3CompliantParams{
		namespace:     ns,
		bucket:        bucket,
		endpoint:      endpoint,
		prefix:        prefix,
		region:        region,
		accessKey:     accessKey,
		secretKey:     secretKey,
		skipSSLVerify: skipSSLVerify,
	}, nil
}

func constructS3CompliantProfile(s3P *s3CompliantParams, secret *v1.Secret) *v1alpha1.Profile {
	return &v1alpha1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    s3P.namespace,
			GenerateName: "s3-profile-",
		},
		Location: v1alpha1.Location{
			Type: v1alpha1.LocationTypeS3Compliant,
			S3Compliant: &v1alpha1.S3CompliantLocation{
				Bucket:   s3P.bucket,
				Endpoint: s3P.endpoint,
				Prefix:   s3P.prefix,
				Region:   s3P.region,
			},
		},
		Credential: v1alpha1.Credential{
			Type: v1alpha1.CredentialTypeKeyPair,
			KeyPair: &v1alpha1.KeyPair{
				IDField:     idField,
				SecretField: secretField,
				Secret: v1alpha1.ObjectReference{
					Name:      secret.GetName(),
					Namespace: secret.GetNamespace(),
				},
			},
		},
		SkipSSLVerify: s3P.skipSSLVerify,
	}
}

func constructSecret(s3P *s3CompliantParams) *v1.Secret {
	data := make(map[string]string, 2)
	data[idField] = s3P.accessKey
	data[secretField] = s3P.secretKey

	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("s3-secret-%s", randString(6)),
			Namespace: s3P.namespace,
		},
		StringData: data,
	}
}

func createSecret(ctx context.Context, s *v1.Secret, cli kubernetes.Interface) (*v1.Secret, error) {
	secret, err := cli.CoreV1().Secrets(s.GetNamespace()).Create(s)
	if err != nil {
		return nil, err
	}
	fmt.Printf("secret '%s' created\n", secret.GetName())
	return secret, nil
}

func deleteSecret(ctx context.Context, secret *v1.Secret, cli kubernetes.Interface) error {
	err := cli.CoreV1().Secrets(secret.GetNamespace()).Delete(secret.GetName(), &metav1.DeleteOptions{})
	if err == nil {
		fmt.Printf("secret '%s' deleted\n", secret.GetName())
	}
	return err
}

func printSecret(secret *v1.Secret) error {
	secret.TypeMeta = metav1.TypeMeta{
		Kind:       reflect.TypeOf(*secret).Name(),
		APIVersion: v1.SchemeGroupVersion.String(),
	}
	secYAML, err := yaml.Marshal(secret)
	if err != nil {
		return errors.New("could not convert generated secret to YAML")
	}
	fmt.Printf(string(secYAML))
	return nil
}

func printProfile(profile *v1alpha1.Profile) error {
	profile.TypeMeta = metav1.TypeMeta{
		Kind:       v1alpha1.ProfileResource.Kind,
		APIVersion: v1alpha1.SchemeGroupVersion.String(),
	}
	profYAML, err := yaml.Marshal(profile)
	if err != nil {
		return errors.New("could not convert generated profile to YAML")
	}
	fmt.Printf(string(profYAML))
	return nil
}

func createProfile(ctx context.Context, profile *v1alpha1.Profile, crCli versioned.Interface) error {
	profile, err := crCli.CrV1alpha1().Profiles(profile.GetNamespace()).Create(profile)
	if err == nil {
		fmt.Printf("profile '%s' created\n", profile.GetName())
	}
	return err
}

func performProfileValidation(p *validateParams) error {
	ctx := context.Background()
	cli, crCli, err := initializeClients()
	if err != nil {
		return errors.Wrap(err, "could not initialize clients for validation")
	}
	prof, err := getProfileFromCmd(ctx, crCli, p)
	if err != nil {
		return err
	}

	return validateProfile(ctx, prof, cli, p.schemaValidationOnly, false)
}

func validateProfile(ctx context.Context, profile *v1alpha1.Profile, cli kubernetes.Interface, schemaValidationOnly bool, printFailStageOnly bool) error {
	var err error
	if err = validate.ProfileSchema(profile); err != nil {
		printStage(schemaValidation, fail)
		return err
	}
	if !printFailStageOnly {
		printStage(schemaValidation, pass)
	}

	for _, d := range []string{regionValidation, readAccessValidation, writeAccessValidation} {
		if schemaValidationOnly {
			if !printFailStageOnly {
				printStage(d, skip)
			}
			continue
		}
		switch d {
		case regionValidation:
			err = validate.ProfileBucket(ctx, profile)
		case readAccessValidation:
			err = validate.ReadAccess(ctx, profile, cli)
		case writeAccessValidation:
			err = validate.WriteAccess(ctx, profile, cli)
		}
		if err != nil {
			printStage(d, fail)
			return err
		}
		if !printFailStageOnly {
			printStage(d, pass)
		}
	}
	if !printFailStageOnly {
		printStage(fmt.Sprintf("All checks passed.. %s\n", pass), "")
	}
	return nil
}

func getProfileFromCmd(ctx context.Context, crCli versioned.Interface, p *validateParams) (*v1alpha1.Profile, error) {
	if p.name != "" {
		return crCli.CrV1alpha1().Profiles(p.namespace).Get(p.name, metav1.GetOptions{})
	}
	return getProfileFromFile(ctx, p.filename)
}

func getProfileFromFile(ctx context.Context, filename string) (*v1alpha1.Profile, error) {
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
	prof := &v1alpha1.Profile{}
	err = d.Decode(prof)
	if err != nil {
		return nil, err
	}
	return prof, nil
}
