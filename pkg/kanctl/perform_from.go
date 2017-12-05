package kanctl

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	crclientv1alpha1 "github.com/kanisterio/kanister/pkg/client/clientset/versioned/typed/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
)

const sourceFlagName = "source"

func newPerformFromCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "from <action> <parent ActionSet>",
		Short: "Perform an action on the artifacts from <parent>",
		Args:  cobra.ExactArgs(2),
		RunE: func(c *cobra.Command, args []string) error {
			return runPerformFrom(c, args)
		},
	}
}

func runPerformFrom(cmd *cobra.Command, args []string) error {
	if len(args) != 2 {
		return newArgsLengthError("expected 2 arguments. got %#v", args)
	}
	actionName, parentName := args[0], args[1]
	ns, err := resolveNamespace(cmd)
	if err != nil {
		return err
	}
	ctx := context.Background()
	return performFrom(ctx, cmd, ns, actionName, parentName)
}

func performFrom(ctx context.Context, cmd *cobra.Command, namespace string, actionName string, parentName string) error {
	config, err := kube.LoadConfig()
	if err != nil {
		return err
	}
	cli, err := crclientv1alpha1.NewForConfig(config)
	if err != nil {
		errors.Wrap(err, "Could not get CRD client")
	}
	pas, err := cli.ActionSets(namespace).Get(parentName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	as, err := childActionSet(actionName, pas)
	if err != nil {
		return err
	}
	return createActionSet(ctx, cmd, cli, namespace, as)
}

func childActionSet(childActionName string, parent *crv1alpha1.ActionSet) (*crv1alpha1.ActionSet, error) {
	if parent.Status == nil || parent.Status.State != crv1alpha1.StateComplete {
		return nil, errors.Errorf("Request parent ActionSet %s has not been exectued", parent.GetName())
	}
	actions := make([]crv1alpha1.ActionSpec, 0, len(parent.Status.Actions))
	for aidx, pa := range parent.Status.Actions {
		actions = append(actions, crv1alpha1.ActionSpec{
			Name:       childActionName,
			Blueprint:  pa.Blueprint,
			Object:     pa.Object,
			Artifacts:  pa.Artifacts,
			Secrets:    parent.Spec.Actions[aidx].Secrets,
			ConfigMaps: parent.Spec.Actions[aidx].ConfigMaps,
		})
	}
	name := fmt.Sprintf("%s-%s-", childActionName, parent.GetName())
	return &crv1alpha1.ActionSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: name,
		},
		Spec: &crv1alpha1.ActionSetSpec{
			Actions: actions,
		},
	}, nil
}
