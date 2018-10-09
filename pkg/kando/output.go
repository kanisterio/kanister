package kando

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/kanisterio/kanister/pkg/output"
)

func newOutputCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "output <key> <value>",
		Short: "Create phase output with given key:value",
		Args: func(c *cobra.Command, args []string) error {
			return validateArguments(c, args)
		},
		// TODO: Example invocations
		RunE: func(c *cobra.Command, args []string) error {
			return runOutputCommand(c, args)
		},
	}
	return cmd
}

func validateArguments(c *cobra.Command, args []string) error {
	if err := cobra.ExactArgs(2); err != nil {
		return errors.New("Command requires exactly two arguments")
	}
	return output.ValidateKey(args[0])
}

func runOutputCommand(c *cobra.Command, args []string) error {
	return output.PrintOutput(args[0], args[1])
}
