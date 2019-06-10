package kanctl

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sYAML "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned"
	"github.com/kanisterio/kanister/pkg/validate"
)

const (
	bucketFlag              = "bucket"
	endpointFlag            = "endpoint"
	prefixFlag              = "prefix"
	regionFlag              = "region"
	awsAccessKeyFlag        = "access-key"
	awsSecretKeyFlag        = "secret-key"
	gcpProjectIDFlag        = "project-id"
	gcpServiceKeyFlag       = "service-key"
	AzureStorageAccountFlag = "storage-account"
	AzureStorageKeyFlag     = "storage-key"

	idField           = "access_key_id"
	secretField       = "secret_access_key"
	skipSSLVerifyFlag = "skip-SSL-verification"

	schemaValidation      = "Validate Profile schema"
	regionValidation      = "Validate bucket region specified in profile"
	readAccessValidation  = "Validate read access to bucket specified in profile"
	writeAccessValidation = "Validate write access to bucket specified in profile"

	secretFormat = "%s-secret-%s"
)

type locationParams struct {
	locationType  v1alpha1.LocationType
	profileName   string
	namespace     string
	bucket        string
	endpoint      string
	prefix        string
	region        string
	skipSSLVerify bool
}

func newProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Create a new profile",
		Args:  cobra.ExactArgs(0),
	}

	cmd.AddCommand(newS3CompliantProfileCmd())
	cmd.AddCommand(newGCPProfileCmd())
	cmd.AddCommand(newAzureProfileCmd())
	cmd.PersistentFlags().StringP(bucketFlag, "b", "", "object store bucket name")
	cmd.PersistentFlags().StringP(endpointFlag, "e", "", "endpoint URL of the object store bucket")
	cmd.PersistentFlags().StringP(prefixFlag, "p", "", "prefix URL of the object store bucket")
	cmd.PersistentFlags().StringP(regionFlag, "r", "", "region of the object store bucket")
	cmd.PersistentFlags().Bool(skipSSLVerifyFlag, false, "if set, SSL verification is disabled for the profile")
	return cmd
}

func newS3CompliantProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "s3compliant",
		Short: "Create new S3 compliant profile",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return createNewProfile(cmd, args)
		},
	}

	cmd.Flags().StringP(awsAccessKeyFlag, "a", "", "access key of the s3 compliant bucket")
	cmd.Flags().StringP(awsSecretKeyFlag, "s", "", "secret key of the s3 compliant bucket")

	cmd.MarkFlagRequired(awsAccessKeyFlag)
	cmd.MarkFlagRequired(awsSecretKeyFlag)
	return cmd
}

func newGCPProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gcp",
		Short: "Create new gcp profile",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return createNewProfile(cmd, args)
		},
	}

	cmd.Flags().StringP(gcpProjectIDFlag, "a", "", "Project ID of the google application")
	cmd.Flags().StringP(gcpServiceKeyFlag, "s", "", "Path to json file containing google application credentials")

	cmd.MarkFlagRequired(gcpProjectIDFlag)
	cmd.MarkFlagRequired(gcpServiceKeyFlag)
	return cmd
}

func newAzureProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "azure",
		Short: "Create new azure profile",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return createNewProfile(cmd, args)
		},
	}

	cmd.Flags().StringP(AzureStorageAccountFlag, "a", "", "Storage account name of the azure storage")
	cmd.Flags().StringP(AzureStorageKeyFlag, "s", "", "Storage account key of the azure storage")

	cmd.MarkFlagRequired(AzureStorageAccountFlag)
	cmd.MarkFlagRequired(AzureStorageKeyFlag)
	return cmd
}

func createNewProfile(cmd *cobra.Command, args []string) error {
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
	lP, err := getLocationParams(cmd)
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true
	secret, err := constructSecret(ctx, lP, cmd)
	if err != nil {
		return err
	}
	profile := constructProfile(lP, secret)
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

func getLocationParams(cmd *cobra.Command) (*locationParams, error) {
	var lType v1alpha1.LocationType
	var profileName string
	ns, err := resolveNamespace(cmd)
	if err != nil {
		return nil, err
	}
	// Location
	bucket, _ := cmd.Flags().GetString(bucketFlag)
	endpoint, _ := cmd.Flags().GetString(endpointFlag)
	prefix, _ := cmd.Flags().GetString(prefixFlag)
	region, _ := cmd.Flags().GetString(regionFlag)

	switch cmd.Name() {
	case "s3compliant":
		lType = v1alpha1.LocationTypeS3Compliant
		profileName = "s3-profile-"
	case "gcp":
		lType = v1alpha1.LocationTypeGCS
		profileName = "gcp-profile-"
	case "azure":
		lType = v1alpha1.LocationTypeAzure
		profileName = "azure-profile-"
	default:
		return nil, errors.New("Profile type not supported: " + cmd.Name())
	}
	skipSSLVerify, _ := cmd.Flags().GetBool(skipSSLVerifyFlag)
	return &locationParams{
		locationType:  lType,
		profileName:   profileName,
		namespace:     ns,
		bucket:        bucket,
		endpoint:      endpoint,
		prefix:        prefix,
		region:        region,
		skipSSLVerify: skipSSLVerify,
	}, nil
}

func constructProfile(lP *locationParams, secret *v1.Secret) *v1alpha1.Profile {
	return &v1alpha1.Profile{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    lP.namespace,
			GenerateName: lP.profileName,
		},
		Location: v1alpha1.Location{
			Type:     lP.locationType,
			Bucket:   lP.bucket,
			Endpoint: lP.endpoint,
			Prefix:   lP.prefix,
			Region:   lP.region,
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
		SkipSSLVerify: lP.skipSSLVerify,
	}
}

func constructSecret(ctx context.Context, lP *locationParams, cmd *cobra.Command) (*v1.Secret, error) {
	data := make(map[string]string, 2)
	secretname := ""
	switch lP.locationType {
	case v1alpha1.LocationTypeS3Compliant:
		accessKey, _ := cmd.Flags().GetString(awsAccessKeyFlag)
		secretKey, _ := cmd.Flags().GetString(awsSecretKeyFlag)
		data[idField] = accessKey
		data[secretField] = secretKey
		secretname = "s3"
	case v1alpha1.LocationTypeGCS:
		projectID, _ := cmd.Flags().GetString(gcpProjectIDFlag)
		filePath, _ := cmd.Flags().GetString(gcpServiceKeyFlag)
		serviceKey, err := getServiceKey(ctx, filePath)
		if err != nil {
			return nil, err
		}
		data[idField] = projectID
		data[secretField] = serviceKey
		secretname = "gcp"
	case v1alpha1.LocationTypeAzure:
		storageAccount, _ := cmd.Flags().GetString(AzureStorageAccountFlag)
		storageKey, _ := cmd.Flags().GetString(AzureStorageKeyFlag)
		data[idField] = storageAccount
		data[secretField] = storageKey
		secretname = "azure"
	}
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf(secretFormat, secretname, randString(6)),
			Namespace: lP.namespace,
		},
		StringData: data,
	}, nil
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

	if profile.Location.Bucket != "" {
		for _, d := range []string{regionValidation, readAccessValidation, writeAccessValidation} {
			if schemaValidationOnly {
				if !printFailStageOnly {
					printStage(d, skip)
				}
				continue
			}
			switch d {
			case regionValidation:
				err = validate.ProfileBucket(ctx, profile, cli)
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

func getServiceKey(ctx context.Context, filename string) (string, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	//Parse the service key
	_, err = google.CredentialsFromJSON(ctx, b, compute.ComputeScope)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
