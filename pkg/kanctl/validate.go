package kanctl

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/client/clientset/versioned"
	"github.com/kanisterio/kanister/pkg/kube"
)

type params struct {
	resourceKind         string
	name                 string
	filename             string
	namespace            string
	schemaValidationOnly bool
}

type indicator string

const (
	fail indicator = `❌`
	pass indicator = `✅`
	skip indicator = `🚫`
)

const (
	nameFlag                 = "name"
	resourceNamespaceFlag    = "resource-namespace"
	filenameFlag             = "filename"
	schemaValidationOnlyFlag = "schema-validation-only"
)

func newValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate <resource>",
		Short: "Validate custom Kanister resources",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return performValidation(cmd, args)
		},
	}
	cmd.Flags().String(nameFlag, "", "specify the K8s name of the custom resource to validate")
	cmd.Flags().StringP(filenameFlag, "f", "", "yaml or json file of the custom resource to validate")
	cmd.Flags().String(resourceNamespaceFlag, "default", "namespace of the custom resource. Used when validating resource specified using --name.")
	cmd.Flags().Bool(schemaValidationOnlyFlag, false, "if set, only schema of resource will be validated")
	return cmd
}

func initializeClients() (kubernetes.Interface, versioned.Interface, error) {
	config, err := kube.LoadConfig()
	if err != nil {
		return nil, nil, err
	}
	cli, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not get the kubernetes client")
	}
	crCli, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not get the CRD client")
	}
	return cli, crCli, nil
}

func performValidation(cmd *cobra.Command, args []string) error {
	p, err := extractParams(cmd, args)
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true

	switch p.resourceKind {
	case "profile":
		return performProfileValidation(p)
	default:
		return errors.Errorf("expected profile.. got %s. Not supported", p.resourceKind)
	}
}

func extractParams(cmd *cobra.Command, args []string) (*params, error) {
	if len(args) != 1 {
		return nil, newArgsLengthError("expected 1 argument. got %#v", args)
	}
	resourceKind := args[0]
	name, _ := cmd.Flags().GetString(nameFlag)
	filename, _ := cmd.Flags().GetString(filenameFlag)
	if name == "" && filename == "" {
		return nil, errors.New("neither name nor filename specified")
	}
	rns, _ := cmd.Flags().GetString(resourceNamespaceFlag)
	schemaValidationOnly, _ := cmd.Flags().GetBool(schemaValidationOnlyFlag)
	return &params{
		resourceKind:         resourceKind,
		name:                 name,
		filename:             filename,
		namespace:            rns,
		schemaValidationOnly: schemaValidationOnly,
	}, nil
}

func printStage(description string, i indicator) {
	switch i {
	case pass:
		fmt.Fprintf(os.Stdout, "Passed the '%s' check.. %s\n", description, i)
	case skip:
		fmt.Fprintf(os.Stdout, "Skipping the '%s' check.. %s\n", description, i)
	case fail:
		fmt.Fprintf(os.Stdout, "Failed the '%s' check.. %s\n", description, i)
	default:
		fmt.Fprintf(os.Stdout, description)
	}
}
