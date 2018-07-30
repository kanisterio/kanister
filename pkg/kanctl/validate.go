package kanctl

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type validateParams struct {
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
	filenameFlag             = "filename"
	resourceNamespaceFlag    = "resource-namespace"
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

func performValidation(cmd *cobra.Command, args []string) error {
	p, err := extractValidateParams(cmd, args)
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

func extractValidateParams(cmd *cobra.Command, args []string) (*validateParams, error) {
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
	return &validateParams{
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
		fmt.Printf("Passed the '%s' check.. %s\n", description, i)
	case skip:
		fmt.Printf("Skipping the '%s' check.. %s\n", description, i)
	case fail:
		fmt.Printf("Failed the '%s' check.. %s\n", description, i)
	default:
		fmt.Printf(description)
	}
}
