package kanctl

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	crclientv1alpha1 "github.com/kanisterio/kanister/pkg/client/clientset/versioned/typed/cr/v1alpha1"
)

func newPerformCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "perform",
		Short: "Create an ActionSet to perform an action",
	}
	cmd.AddCommand(newPerformFromCommand())
	return cmd
}

func createActionSet(ctx context.Context, cmd *cobra.Command, cli crclientv1alpha1.CrV1alpha1Interface, namespace string, as *crv1alpha1.ActionSet) error {
	as, err := cli.ActionSets(namespace).Create(as)
	if err == nil {
		cmd.Println(fmt.Sprintf("actionset %s created", as.Name))
	}
	return err
}
