package kanctl

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	crclientv1alpha1 "github.com/kanisterio/kanister/pkg/client/clientset/versioned/typed/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
)

const (
	sourceFlagName  = "from"
	optionsFlagName = "options"
)

func newPerformFromCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "perform <action>",
		Short: "Perform an action on the artifacts from <parent>",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runPerformFrom(c, args)
		},
	}
	cmd.PersistentFlags().StringP(sourceFlagName, "f", "", "specify name of the action set(required)")
	cmd.Flags().StringSliceP(optionsFlagName, "o", []string{}, "specify options for the action set, comma separated key=value pairs (eg: --options key1=value1,key2=value2)")
	cmd.MarkFlagRequired(sourceFlagName)
	return cmd

}

func runPerformFrom(cmd *cobra.Command, args []string) error {
	parentName := cmd.Flag(sourceFlagName).Value.String()
	if len(args) != 1 {
		return newArgsLengthError("expected 1 argument. got %#v", args)
	}
	actionName := args[0]
	ns, err := resolveNamespace(cmd)
	if err != nil {
		return err
	}
	options, err := parseOptions(cmd)
	if err != nil {
		return err
	}

	ctx := context.Background()
	return performFrom(ctx, cmd, ns, actionName, parentName, options)
}

func performFrom(ctx context.Context, cmd *cobra.Command, namespace string, actionName string, parentName string, options map[string]string) error {
	config, err := kube.LoadConfig()
	if err != nil {
		return err
	}
	cli, err := crclientv1alpha1.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "Could not get CRD client")
	}
	pas, err := cli.ActionSets(namespace).Get(parentName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	as, err := childActionSet(actionName, pas, options)
	if err != nil {
		return err
	}
	return createActionSet(ctx, cmd, cli, namespace, as)
}

func childActionSet(childActionName string, parent *crv1alpha1.ActionSet, options map[string]string) (*crv1alpha1.ActionSet, error) {
	if parent.Status == nil || parent.Status.State != crv1alpha1.StateComplete {
		return nil, errors.Errorf("Request parent ActionSet %s has not been executed", parent.GetName())
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
			Profile:    parent.Spec.Actions[aidx].Profile,
			Options:    mergeOptions(options, parent.Spec.Actions[aidx].Options),
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

func createActionSet(ctx context.Context, cmd *cobra.Command, cli crclientv1alpha1.CrV1alpha1Interface, namespace string, as *crv1alpha1.ActionSet) error {
	as, err := cli.ActionSets(namespace).Create(as)
	if err == nil {
		cmd.Println(fmt.Sprintf("actionset %s created", as.Name))
	}
	return err
}

func parseOptions(cmd *cobra.Command) (map[string]string, error) {
	optionsFromCmd, _ := cmd.Flags().GetStringSlice(optionsFlagName)
	options := make(map[string]string)

	for _, kv := range optionsFromCmd {
		// Cobra takes care of trimming spaces
		kvPair := strings.Split(kv, "=")
		if len(kvPair) != 2 {
			return nil, errors.Errorf("Expected options as key=value pairs. Got %s", kv)
		}
		options[kvPair[0]] = kvPair[1]
	}
	return options, nil
}

func mergeOptions(src map[string]string, dst map[string]string) map[string]string {
	final := make(map[string]string, len(src)+len(dst))
	for k, v := range dst {
		final[k] = v
	}
	// Override default options and set additional ones
	for k, v := range src {
		final[k] = v
	}
	return final
}
