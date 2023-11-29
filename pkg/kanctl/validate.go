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

package kanctl

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	kanister "github.com/kanisterio/kanister/pkg"
)

type validateParams struct {
	resourceKind         string
	name                 string
	filename             string
	namespace            string
	schemaValidationOnly bool
	functionVersion      string
}

const (
	nameFlag                 = "name"
	filenameFlag             = "filename"
	funcVersionFlag          = "functionVersion"
	resourceNamespaceFlag    = "resource-namespace"
	schemaValidationOnlyFlag = "schema-validation-only"
)

func newValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate <resource>",
		Short: "Validate custom Kanister resources",
		Args:  cobra.ExactArgs(1),
		RunE:  performValidation,
	}
	cmd.Flags().String(nameFlag, "", "specify the K8s name of the custom resource to validate")
	cmd.Flags().StringP(filenameFlag, "f", "", "yaml or json file of the custom resource to validate")
	cmd.Flags().String(resourceNamespaceFlag, "default", "namespace of the custom resource. Used when validating resource specified using --name.")
	cmd.Flags().Bool(schemaValidationOnlyFlag, false, "if set, only schema of resource will be validated")
	cmd.Flags().StringP(funcVersionFlag, "v", kanister.DefaultVersion, "kanister function version, e.g., v0.0.0")
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
	case "blueprint":
		return performBlueprintValidation(p)
	case "repository-server-secrets":
		return performRepoServerSecretsValidation(cmd.Context(), p)
	default:
		return errors.Errorf("resource %s is not supported for validate subcommand", p.resourceKind)
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
	funcVersion, _ := cmd.Flags().GetString(funcVersionFlag)

	return &validateParams{
		resourceKind:         resourceKind,
		name:                 name,
		filename:             filename,
		namespace:            rns,
		schemaValidationOnly: schemaValidationOnly,
		functionVersion:      funcVersion,
	}, nil
}
