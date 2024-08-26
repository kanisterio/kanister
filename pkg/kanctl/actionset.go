// Copyright 2024 The Kanister Authors.
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
	"regexp"
	"strings"
	"sync"

	osversioned "github.com/openshift/client-go/apps/clientset/versioned"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/client/clientset/versioned"
	"github.com/kanisterio/kanister/pkg/function"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/poll"
)

const (
	actionFlagName                       = "action"
	actionSetFlagName                    = "name"
	blueprintFlagName                    = "blueprint"
	configMapsFlagName                   = "config-maps"
	deploymentFlagName                   = "deployment"
	optionsFlagName                      = "options"
	profileFlagName                      = "profile"
	repositoryServerFlagName             = "repository-server"
	pvcFlagName                          = "pvc"
	secretsFlagName                      = "secrets"
	statefulSetFlagName                  = "statefulset"
	deploymentConfigFlagName             = "deploymentconfig"
	sourceFlagName                       = "from"
	selectorFlagName                     = "selector"
	selectorKindFlag                     = "kind"
	selectorNamespaceFlag                = "selector-namespace"
	namespaceTargetsFlagName             = "namespacetargets"
	objectsFlagName                      = "objects"
	waitForRepositoryServerReadyFlagName = "wait-for-repository-server"
	labelsFlagName                       = "labels"
	podAnnotationsFlagName               = "podannotations"
	podLabelsFlagName                    = "podlabels"
)

var (
	errMissingFieldActionName = fmt.Errorf("missing action name. use the --action flag to specify the action name")
	errInvalidFieldLabels     = fmt.Errorf("invalid --labels value. make sure the value for field --labels is correct")
)

type PerformParams struct {
	Namespace        string
	ActionName       string
	ActionSetName    string
	ParentName       string
	Blueprint        string
	DryRun           bool
	Objects          []crv1alpha1.ObjectReference
	Options          map[string]string
	Profile          *crv1alpha1.ObjectReference
	RepositoryServer *crv1alpha1.ObjectReference
	Secrets          map[string]crv1alpha1.ObjectReference
	ConfigMaps       map[string]crv1alpha1.ObjectReference
	Labels           map[string]string
	PodLabels        map[string]string
	PodAnnotations   map[string]string
}

func newActionSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "actionset",
		Short: "Create a new ActionSet or override a <parent> ActionSet",
		Args:  cobra.ExactArgs(0),
		RunE:  initializeAndPerform,
	}
	cmd.Flags().StringP(sourceFlagName, "f", "", "specify name of the action set")

	cmd.Flags().StringP(actionFlagName, "a", "", "action for the action set (required if creating a new action set)")
	cmd.Flags().StringP(actionSetFlagName, "A", "", "name of the new actionset (optional. if not specified, kanctl will generate one based on the action name")
	cmd.Flags().StringP(blueprintFlagName, "b", "", "blueprint for the action set (required if creating a new action set)")
	cmd.Flags().StringSliceP(configMapsFlagName, "c", []string{}, "config maps for the action set, comma separated ref=namespace/name pairs (eg: --config-maps ref1=namespace1/name1,ref2=namespace2/name2)")
	cmd.Flags().StringSliceP(deploymentFlagName, "d", []string{}, "deployment for the action set, comma separated namespace/name pairs (eg: --deployment namespace1/name1,namespace2/name2)")
	cmd.Flags().StringSliceP(optionsFlagName, "o", []string{}, "specify options for the action set, comma separated key=value pairs (eg: --options key1=value1,key2=value2)")
	cmd.Flags().StringP(profileFlagName, "p", "", "profile for the action set")
	cmd.Flags().StringP(repositoryServerFlagName, "r", "", "kopia repository server custom resource reference (eg: --repository-server namespace/name)")
	cmd.Flags().StringSliceP(pvcFlagName, "v", []string{}, "pvc for the action set, comma separated namespace/name pairs (eg: --pvc namespace1/name1,namespace2/name2)")
	cmd.Flags().StringSliceP(secretsFlagName, "s", []string{}, "secrets for the action set, comma separated ref=namespace/name pairs (eg: --secrets ref1=namespace1/name1,ref2=namespace2/name2)")
	cmd.Flags().StringSliceP(statefulSetFlagName, "t", []string{}, "statefulset for the action set, comma separated namespace/name pairs (eg: --statefulset namespace1/name1,namespace2/name2)")
	cmd.Flags().StringSliceP(deploymentConfigFlagName, "D", []string{}, "deploymentconfig for action set, comma separated namespace/name pairs "+
		"(e.g. --deploymentconfig namespace1/name1,namespace2/name2). Will ideally be used on openshift clusters.")
	cmd.Flags().StringP(selectorFlagName, "l", "", "k8s selector for objects")
	cmd.Flags().StringP(selectorKindFlag, "k", "all", "resource kind to apply selector on. Used along with the selector specified using --selector/-l")
	cmd.Flags().String(selectorNamespaceFlag, "", "namespace to apply selector on. Used along with the selector specified using --selector/-l")
	cmd.Flags().StringSliceP(namespaceTargetsFlagName, "T", []string{}, "namespaces for the action set, comma separated list of namespaces (eg: --namespacetargets namespace1,namespace2)")
	cmd.Flags().StringSliceP(objectsFlagName, "O", []string{}, "objects for the action set, comma separated list of object references (eg: --objects group/version/resource/namespace1/name1,group/version/resource/namespace2/name2)")
	cmd.Flags().BoolP(waitForRepositoryServerReadyFlagName, "w", false, "wait for repository server to be ready before creating actionset")
	cmd.Flags().String(labelsFlagName, "", "Labels that should be added to the created actionset, space chars would be trimmed automatically. Multiple labels can be separate by comma(,) (eg: --labels key=value,foo=bar)")
	cmd.Flags().StringToString(podAnnotationsFlagName, nil, "This flag can be used to configure annotations of the pods that are created by Kanister functions that are run by this ActionSet. (eg. --podannotations=key1=value1,key2=value2)")
	cmd.Flags().StringToString(podLabelsFlagName, nil, "This flag can be used to configure labels of the pods that are created by Kanister functions that are run by this ActionSet. (eg: --podlabels=key1=value1,key2=value2)")
	return cmd
}

func initializeAndPerform(cmd *cobra.Command, args []string) error {
	cli, crCli, osCli, err := initializeClients()
	if err != nil {
		return err
	}
	params, err := extractPerformParams(cmd, args, cli, osCli)
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true
	ctx := context.Background()
	valFlag, _ := cmd.Flags().GetBool(skipValidationFlag)
	if !valFlag {
		repoServerReady, _ := cmd.Flags().GetBool(waitForRepositoryServerReadyFlagName)
		err = verifyParams(ctx, params, cli, crCli, osCli, repoServerReady)
		if err != nil {
			return err
		}
	}
	err = isDataMoverProvided(cmd)
	if err != nil {
		fmt.Printf("Warning: %s\n", err.Error())
	}
	return perform(ctx, crCli, params)
}

func perform(ctx context.Context, crCli versioned.Interface, params *PerformParams) error {
	var as, pas *crv1alpha1.ActionSet
	var err error

	switch {
	case params.ParentName != "":
		pas, err = crCli.CrV1alpha1().ActionSets(params.Namespace).Get(ctx, params.ParentName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		as, err = ChildActionSet(pas, params)
	case len(params.Objects) > 0:
		as, err = newActionSet(params)
	default:
		return errors.New("no objects found to perform action set. Please pass a valid parent action set and/or selector")
	}
	if err != nil {
		return err
	}
	if params.DryRun {
		return printActionSet(as)
	}
	return createActionSet(ctx, crCli, params.Namespace, as)
}

func newActionSet(params *PerformParams) (*crv1alpha1.ActionSet, error) {
	if params.ActionName == "" {
		return nil, errors.New("action required to create new action set")
	}
	if params.Blueprint == "" {
		return nil, errors.New("blueprint required to create new action set")
	}
	actions := make([]crv1alpha1.ActionSpec, 0, len(params.Objects))
	for _, obj := range params.Objects {
		actions = append(actions, crv1alpha1.ActionSpec{
			Name:             params.ActionName,
			Blueprint:        params.Blueprint,
			Object:           obj,
			Secrets:          params.Secrets,
			ConfigMaps:       params.ConfigMaps,
			Profile:          params.Profile,
			RepositoryServer: params.RepositoryServer,
			Options:          params.Options,
			PodAnnotations:   params.PodAnnotations,
			PodLabels:        params.PodLabels,
		})
	}

	name, err := generateActionSetName(params)
	if err != nil {
		return nil, err
	}

	actionset := &crv1alpha1.ActionSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: &crv1alpha1.ActionSetSpec{
			Actions: actions,
		},
	}
	if params.Labels != nil {
		actionset.Labels = params.Labels
	}

	return actionset, nil
}

func ChildActionSet(parent *crv1alpha1.ActionSet, params *PerformParams) (*crv1alpha1.ActionSet, error) {
	if parent.Status == nil || parent.Status.State != crv1alpha1.StateComplete {
		return nil, errors.Errorf("Request parent ActionSet %s has not been executed", parent.GetName())
	}

	actions := make([]crv1alpha1.ActionSpec, 0, len(parent.Status.Actions)*max(1, len(params.Objects)))
	for aidx, pa := range parent.Status.Actions {
		as := crv1alpha1.ActionSpec{
			Name:             parent.Spec.Actions[aidx].Name,
			Blueprint:        pa.Blueprint,
			Object:           pa.Object,
			Artifacts:        pa.Artifacts,
			Secrets:          parent.Spec.Actions[aidx].Secrets,
			ConfigMaps:       parent.Spec.Actions[aidx].ConfigMaps,
			Profile:          parent.Spec.Actions[aidx].Profile,
			RepositoryServer: parent.Spec.Actions[aidx].RepositoryServer,
			Options:          mergeOptions(params.Options, parent.Spec.Actions[aidx].Options),
			PodAnnotations:   params.PodAnnotations,
			PodLabels:        params.PodLabels,
		}
		// Apply overrides
		if params.ActionName != "" {
			as.Name = params.ActionName
		}
		if params.Blueprint != "" {
			as.Blueprint = params.Blueprint
		}
		if len(params.Secrets) > 0 {
			as.Secrets = params.Secrets
		}
		if len(params.ConfigMaps) > 0 {
			as.ConfigMaps = params.ConfigMaps
		}
		if params.Profile != nil {
			as.Profile = params.Profile
		}
		if params.RepositoryServer != nil {
			as.RepositoryServer = params.RepositoryServer
		}
		if len(params.Objects) > 0 {
			for _, obj := range params.Objects {
				asCopy := as.DeepCopy()
				asCopy.Object = obj

				actions = append(actions, *asCopy)
			}
		} else {
			actions = append(actions, as)
		}
	}

	name, err := generateActionSetName(params)
	if err != nil {
		return nil, err
	}

	actionset := &crv1alpha1.ActionSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: &crv1alpha1.ActionSetSpec{
			Actions: actions,
		},
	}
	if params.Labels != nil {
		actionset.Labels = params.Labels
	}

	return actionset, nil
}

func createActionSet(ctx context.Context, crCli versioned.Interface, namespace string, as *crv1alpha1.ActionSet) error {
	as, err := crCli.CrV1alpha1().ActionSets(namespace).Create(ctx, as, metav1.CreateOptions{})
	if err == nil {
		fmt.Printf("actionset %s created\n", as.Name)
	}
	return err
}

func printActionSet(as *crv1alpha1.ActionSet) error {
	as.TypeMeta = metav1.TypeMeta{
		Kind:       crv1alpha1.ActionSetResource.Kind,
		APIVersion: crv1alpha1.SchemeGroupVersion.String(),
	}
	asYAML, err := yaml.Marshal(as)
	if err != nil {
		return errors.New("could not convert generated action set to YAML")
	}
	fmt.Printf("%s", asYAML)
	return nil
}

func extractPerformParams(cmd *cobra.Command, args []string, cli kubernetes.Interface, osCli osversioned.Interface) (*PerformParams, error) {
	if len(args) != 0 {
		return nil, newArgsLengthError("expected 0 arguments. got %#v", args)
	}
	ns, err := resolveNamespace(cmd)
	if err != nil {
		return nil, err
	}
	actionName, _ := cmd.Flags().GetString(actionFlagName)
	actionSetName, _ := cmd.Flags().GetString(actionSetFlagName)
	parentName, _ := cmd.Flags().GetString(sourceFlagName)
	blueprint, _ := cmd.Flags().GetString(blueprintFlagName)
	dryRun, _ := cmd.Flags().GetBool(dryRunFlag)
	labels, _ := cmd.Flags().GetString(labelsFlagName)
	profile, err := parseProfile(cmd, ns)
	if err != nil {
		return nil, err
	}
	repositoryServer, err := parseRepositoryServer(cmd)
	if err != nil {
		return nil, err
	}
	cms, err := parseConfigMaps(cmd)
	if err != nil {
		return nil, err
	}
	objects, err := parseObjects(cmd, cli, osCli)
	if err != nil {
		return nil, err
	}
	options, err := parseOptions(cmd)
	if err != nil {
		return nil, err
	}
	secrets, err := parseSecrets(cmd)
	if err != nil {
		return nil, err
	}
	ls, err := parseLabels(labels)
	if err != nil {
		return nil, err
	}

	podAnnotations, _ := cmd.Flags().GetStringToString(podAnnotationsFlagName)
	podLabels, _ := cmd.Flags().GetStringToString(podLabelsFlagName)

	err = function.ValidateAnnotations(podAnnotations)
	if err != nil {
		return nil, err
	}

	err = function.ValidateLabels(podLabels)
	if err != nil {
		return nil, err
	}

	return &PerformParams{
		Namespace:        ns,
		ActionName:       actionName,
		ActionSetName:    actionSetName,
		ParentName:       parentName,
		Blueprint:        blueprint,
		DryRun:           dryRun,
		Objects:          objects,
		Options:          options,
		Secrets:          secrets,
		ConfigMaps:       cms,
		Profile:          profile,
		RepositoryServer: repositoryServer,
		Labels:           ls,
		PodAnnotations:   podAnnotations,
		PodLabels:        podLabels,
	}, nil
}

// parseLabels parses the given string to labels, it does some basic
// validations, for example the key and value must be separated by `=`
// and neither key nor value can have `,` in it.
// If there are some issue even after this, that would be caught by apiserver
// when actionset is actually created.
func parseLabels(label string) (map[string]string, error) {
	if label == "" {
		return nil, nil
	}
	labels := strings.Split(label, ",")

	parsed := map[string]string{}
	for _, l := range labels {
		if !strings.Contains(l, "=") {
			return nil, errInvalidFieldLabels
		}

		kv := strings.Split(l, "=")
		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		if strings.HasPrefix(key, ",") ||
			strings.HasSuffix(key, ",") ||
			strings.HasPrefix(value, ",") ||
			strings.HasSuffix(value, ",") {
			return nil, errInvalidFieldLabels
		}

		parsed[key] = value
	}
	return parsed, nil
}

func parseConfigMaps(cmd *cobra.Command) (map[string]crv1alpha1.ObjectReference, error) {
	configMapsFromCmd, _ := cmd.Flags().GetStringSlice(configMapsFlagName)
	cms, err := parseReferences(configMapsFromCmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse config maps")
	}
	return cms, nil
}

func parseProfile(cmd *cobra.Command, ns string) (*crv1alpha1.ObjectReference, error) {
	profileName, _ := cmd.Flags().GetString(profileFlagName)
	if profileName == "" {
		return nil, nil
	}
	if strings.Contains(profileName, "/") {
		temp := strings.Split(profileName, "/")
		if len(temp) != 2 {
			return nil, errors.Errorf("Invalid profile name %s, should be of the form ( --profile namespace/name OR --profile name)", profileName)
		}
		ns = temp[0]
		profileName = temp[1]
	}
	return &crv1alpha1.ObjectReference{
		Name:      profileName,
		Namespace: ns,
	}, nil
}

// parseRepositoryServer returns the object reference
// for the passed Repository Server
func parseRepositoryServer(cmd *cobra.Command) (*crv1alpha1.ObjectReference, error) {
	repositoryServerName, _ := cmd.Flags().GetString(repositoryServerFlagName)
	if repositoryServerName == "" {
		return nil, nil
	}
	if strings.Contains(repositoryServerName, "/") {
		nsName := strings.Split(repositoryServerName, "/")
		if len(nsName) != 2 {
			return nil, errors.Errorf("Invalid repository server name %s, it should be of the form ( --repository-server namespace/name )", repositoryServerName)
		}
		ns := nsName[0]
		repositoryServerName = nsName[1]
		return &crv1alpha1.ObjectReference{
			Name:      repositoryServerName,
			Namespace: ns,
		}, nil
	}
	return nil, errors.Errorf("Invalid repository server name %s, it should be of the form ( --repository-server namespace/name )", repositoryServerName)
}

func parseSecrets(cmd *cobra.Command) (map[string]crv1alpha1.ObjectReference, error) {
	secretsFromCmd, _ := cmd.Flags().GetStringSlice(secretsFlagName)
	secrets, err := parseReferences(secretsFromCmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse secrets")
	}
	return secrets, nil
}

func parseObjects(cmd *cobra.Command, cli kubernetes.Interface, osCli osversioned.Interface) ([]crv1alpha1.ObjectReference, error) {
	var objects []crv1alpha1.ObjectReference
	objs := make(map[string][]string)

	deployments, _ := cmd.Flags().GetStringSlice(deploymentFlagName)
	statefulSets, _ := cmd.Flags().GetStringSlice(statefulSetFlagName)
	deploymentConfig, _ := cmd.Flags().GetStringSlice(deploymentConfigFlagName)
	pvcs, _ := cmd.Flags().GetStringSlice(pvcFlagName)
	namespaces, _ := cmd.Flags().GetStringSlice(namespaceTargetsFlagName)

	objs[param.DeploymentKind] = deployments
	objs[param.StatefulSetKind] = statefulSets
	objs[param.DeploymentConfigKind] = deploymentConfig
	objs[param.PVCKind] = pvcs
	objs[param.NamespaceKind] = namespaces

	parsed := make(map[string]bool)
	fromCmd, err := parseObjectsFromCmd(objs, parsed)
	if err != nil {
		return nil, err
	}
	objects = append(objects, fromCmd...)

	genObjects, _ := cmd.Flags().GetStringSlice(objectsFlagName)
	fromCmd, err = parseGenericObjects(genObjects)
	if err != nil {
		return nil, err
	}
	objects = append(objects, fromCmd...)

	selectorString, _ := cmd.Flags().GetString(selectorFlagName)
	if selectorString != "" {
		// parse selector before making calls to K8s
		selector, err := labels.Parse(selectorString)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse selector")
		}
		kind, _ := cmd.Flags().GetString(selectorKindFlag)
		sns, _ := cmd.Flags().GetString(selectorNamespaceFlag)
		fromSelector, err := parseObjectsFromSelector(selector.String(), kind, sns, cli, osCli, parsed)
		if err != nil {
			return nil, err
		}
		objects = append(objects, fromSelector...)
	}
	return objects, nil
}

func parseObjectsFromCmd(objs map[string][]string, parsed map[string]bool) ([]crv1alpha1.ObjectReference, error) {
	var objects []crv1alpha1.ObjectReference
	for kind, resources := range objs {
		for _, resource := range resources {
			namespace, name, err := parseName(kind, resource)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to parse %s", kind)
			}
			obj := fmt.Sprintf("%s=%s/%s", kind, namespace, name)
			if _, ok := parsed[obj]; ok || obj == "" {
				continue
			}
			parsed[obj] = true
			switch strings.ToLower(kind) {
			case param.DeploymentKind:
				objects = append(objects, crv1alpha1.ObjectReference{Kind: param.DeploymentKind, Namespace: namespace, Name: name})
			case param.StatefulSetKind:
				objects = append(objects, crv1alpha1.ObjectReference{Kind: param.StatefulSetKind, Namespace: namespace, Name: name})
			case param.DeploymentConfigKind:
				objects = append(objects, crv1alpha1.ObjectReference{Kind: param.DeploymentConfigKind, Namespace: namespace, Name: name})
			case param.PVCKind:
				objects = append(objects, crv1alpha1.ObjectReference{Kind: param.PVCKind, Namespace: namespace, Name: name})
			case param.NamespaceKind:
				objects = append(objects, crv1alpha1.ObjectReference{Kind: param.NamespaceKind, Namespace: namespace, Name: name})
			default:
				return nil, errors.Errorf("unsupported or unknown object kind '%s'. Supported %s, %s and %s", kind, param.DeploymentKind, param.StatefulSetKind, param.PVCKind)
			}
		}
	}
	return objects, nil
}

func parseGenericObjects(objs []string) ([]crv1alpha1.ObjectReference, error) {
	var objects []crv1alpha1.ObjectReference
	for _, ref := range objs {
		o, err := parseGenericObjectReference(ref)
		if err != nil {
			return nil, err
		}
		objects = append(objects, o)
	}
	return objects, nil
}

func parseGenericObjectReference(s string) (crv1alpha1.ObjectReference, error) {
	// Looking for group/version/resource/namespace/name
	reg := regexp.MustCompile(`([\w-.]*/)*([\w-.]+)/([\w-.]+)/([\w-.]+)/([\w-.]+)`)
	m := reg.FindStringSubmatch(s)
	if len(m) != 6 && len(m) != 5 {
		return crv1alpha1.ObjectReference{}, errors.Errorf("Expected group/version/resource/namespace/name or in the case of core group's resources version/resource/namespace/name . Got %s %d", s, len(m))
	}
	return crv1alpha1.ObjectReference{
		Group:      m[1],
		APIVersion: m[2],
		Resource:   m[3],
		Namespace:  m[4],
		Name:       m[5],
	}, nil
}

func parseObjectsFromSelector(selector, kind, sns string, cli kubernetes.Interface, osCli osversioned.Interface, parsed map[string]bool) ([]crv1alpha1.ObjectReference, error) { //nolint:gocyclo
	ctx := context.Background()
	var objects []crv1alpha1.ObjectReference
	appendObj := func(kind, namespace, name string) {
		r := fmt.Sprintf("%s=%s/%s", kind, namespace, name)
		if _, ok := parsed[r]; !ok {
			objects = append(objects, crv1alpha1.ObjectReference{Kind: kind, Namespace: namespace, Name: name})
			parsed[r] = true
		}
	}
	switch kind {
	case "all":
		fallthrough
	case param.DeploymentKind:
		dpts, err := cli.AppsV1().Deployments(sns).List(ctx, metav1.ListOptions{LabelSelector: selector})
		if err != nil {
			return nil, errors.Errorf("failed to get deployments using selector '%s' in namespace '%s'", selector, sns)
		}
		for _, d := range dpts.Items {
			appendObj(param.DeploymentKind, d.Namespace, d.Name)
		}
		if kind != "all" {
			break
		}
		fallthrough
	case param.DeploymentConfigKind:
		// use open shift SDK to get the deployment config resource
		dcs, err := osCli.AppsV1().DeploymentConfigs(sns).List(ctx, metav1.ListOptions{LabelSelector: selector})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get deploymentconfig using select '%s' in namespaces '%s'", selector, sns)
		}
		for _, d := range dcs.Items {
			appendObj(param.DeploymentConfigKind, d.Namespace, d.Name)
		}
		if kind != "all" {
			break
		}
		fallthrough
	case param.StatefulSetKind:
		ss, err := cli.AppsV1().StatefulSets(sns).List(ctx, metav1.ListOptions{LabelSelector: selector})
		if err != nil {
			return nil, errors.Errorf("failed to get statefulsets using selector '%s' in namespace '%s'", selector, sns)
		}
		for _, s := range ss.Items {
			appendObj(param.StatefulSetKind, s.Namespace, s.Name)
		}
		if kind != "all" {
			break
		}
		fallthrough
	case param.PVCKind:
		pvcs, err := cli.CoreV1().PersistentVolumeClaims(sns).List(ctx, metav1.ListOptions{LabelSelector: selector})
		if err != nil {
			return nil, errors.Errorf("failed to get pvcs using selector '%s' in namespace '%s'", selector, sns)
		}
		for _, pvc := range pvcs.Items {
			appendObj(param.PVCKind, pvc.Namespace, pvc.Name)
		}
	case param.NamespaceKind:
		namespaces, err := cli.CoreV1().Namespaces().List(ctx, metav1.ListOptions{LabelSelector: selector})
		if err != nil {
			return nil, errors.Errorf("failed to get namespaces using selector '%s' '", selector)
		}
		for _, ns := range namespaces.Items {
			appendObj(param.NamespaceKind, ns.Namespace, ns.Name)
		}
	default:
		return nil, errors.Errorf("unsupported or unknown object kind '%s'. Supported %s, %s and %s", kind, param.DeploymentKind, param.StatefulSetKind, param.PVCKind)
	}
	return objects, nil
}

func parseOptions(cmd *cobra.Command) (map[string]string, error) {
	optionsFromCmd, _ := cmd.Flags().GetStringSlice(optionsFlagName)
	options := make(map[string]string)

	for _, kv := range optionsFromCmd {
		if kv == "" {
			continue
		}
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

func parseReferences(references []string) (map[string]crv1alpha1.ObjectReference, error) {
	m := make(map[string]crv1alpha1.ObjectReference)
	parsed := make(map[string]bool)

	for _, r := range references {
		if _, ok := parsed[r]; ok || r == "" {
			continue
		}
		parsed[r] = true
		ref, namespace, name, err := parseReference(r)
		if err != nil {
			return nil, err
		}
		m[ref] = crv1alpha1.ObjectReference{
			Name:      name,
			Namespace: namespace,
		}
	}
	return m, nil
}

func parseReference(r string) (ref, namespace, name string, err error) {
	reg := regexp.MustCompile(`([\w-.]+)=([\w-.]+)/([\w-.]+)`)
	matches := reg.FindStringSubmatch(r)
	if len(matches) != 4 {
		return "", "", "", errors.Errorf("Expected ref=namespace/name. Got %s", r)
	}
	return matches[1], matches[2], matches[3], nil
}

func parseName(k string, r string) (namespace, name string, err error) {
	if strings.ToLower(k) == param.NamespaceKind {
		return r, r, nil
	}
	reg := regexp.MustCompile(`([\w-.]+)/([\w-.]+)`)
	m := reg.FindStringSubmatch(r)
	if len(m) != 3 {
		return "", "", errors.Errorf("Expected namespace/name. Got %s", r)
	}
	return m[1], m[2], nil
}

func verifyParams(ctx context.Context, p *PerformParams, cli kubernetes.Interface, crCli versioned.Interface, osCli osversioned.Interface, waitForRepoServerReady bool) error {
	const notFoundTmpl = "Please make sure '%s' with name '%s' exists in namespace '%s'"
	msgs := make(chan error)
	wg := sync.WaitGroup{}
	wg.Add(6)

	// Blueprint
	go func() {
		defer wg.Done()
		if p.Blueprint != "" {
			_, err := crCli.CrV1alpha1().Blueprints(p.Namespace).Get(ctx, p.Blueprint, metav1.GetOptions{})
			if err != nil {
				msgs <- errors.Wrapf(err, notFoundTmpl, "blueprint", p.Blueprint, p.Namespace)
			}
		}
	}()

	// Profile
	go func() {
		defer wg.Done()
		if p.Profile != nil {
			_, err := crCli.CrV1alpha1().Profiles(p.Profile.Namespace).Get(ctx, p.Profile.Name, metav1.GetOptions{})
			if err != nil {
				msgs <- errors.Wrapf(err, notFoundTmpl, "profile", p.Profile.Name, p.Profile.Namespace)
			}
		}
	}()

	// RepositoryServer
	go func() {
		defer wg.Done()
		err := verifyRepositoryServerParams(ctx, crCli, p.RepositoryServer, waitForRepoServerReady)
		if err != nil {
			msgs <- err
		}
	}()

	// Objects
	go func() {
		defer wg.Done()
		err := verifyObjectParams(p, cli, osCli, ctx)
		if err != nil {
			msgs <- err
		}
	}()

	// ConfigMaps
	go func() {
		defer wg.Done()
		for _, cm := range p.ConfigMaps {
			_, err := cli.CoreV1().ConfigMaps(cm.Namespace).Get(ctx, cm.Name, metav1.GetOptions{})
			if err != nil {
				msgs <- errors.Wrapf(err, notFoundTmpl, "config map", cm.Name, cm.Namespace)
			}
		}
	}()

	// Secrets
	go func() {
		defer wg.Done()
		for _, secret := range p.Secrets {
			_, err := cli.CoreV1().Secrets(secret.Namespace).Get(ctx, secret.Name, metav1.GetOptions{})
			if err != nil {
				msgs <- errors.Wrapf(err, notFoundTmpl, "secret", secret.Name, secret.Namespace)
			}
		}
	}()

	go func() {
		wg.Wait()
		close(msgs)
	}()

	vFail := false
	for msg := range msgs {
		vFail = true
		fmt.Println(msg)
	}

	if vFail {
		return errors.Errorf("resource verification failed")
	}
	return nil
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func generateActionSetName(p *PerformParams) (string, error) {
	if p.ActionSetName != "" {
		return p.ActionSetName, nil
	}

	if p.ActionName != "" {
		if p.ParentName != "" {
			return fmt.Sprintf("%s-%s-%s", p.ActionName, p.ParentName, rand.String(5)), nil
		}

		return fmt.Sprintf("%s-%s", p.ActionName, rand.String(5)), nil
	}

	if p.ParentName != "" {
		return fmt.Sprintf("%s-%s", p.ParentName, rand.String(5)), nil
	}

	return "", errMissingFieldActionName
}

func verifyRepositoryServerParams(ctx context.Context, crCli versioned.Interface, repoServer *crv1alpha1.ObjectReference, waitForRepoServerReady bool) error {
	if repoServer != nil {
		rs, err := crCli.CrV1alpha1().RepositoryServers(repoServer.Namespace).Get(ctx, repoServer.Name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return errors.Wrapf(err, "Please make sure '%s' with name '%s' exists in namespace '%s'", "repository-server", repoServer.Name, repoServer.Namespace)
			}
			return errors.New("error while fetching repo server")
		}
		if waitForRepoServerReady {
			return waitForKopiaRepositoryServerReady(ctx, crCli, rs)
		}
		if rs.Status.Progress != crv1alpha1.Ready {
			err = errors.New("Repository Server Not Ready")
			return errors.Wrapf(err, "Please make sure that Repository Server CR '%s' is in Ready State", repoServer.Name)
		}
	}
	return nil
}

func waitForKopiaRepositoryServerReady(ctx context.Context, crCli versioned.Interface, rs *crv1alpha1.RepositoryServer) error {
	timeoutCtx, waitCancel := context.WithTimeout(ctx, contextWaitTimeout)
	defer waitCancel()
	pollErr := poll.Wait(timeoutCtx, func(ctx context.Context) (bool, error) {
		repositoryServer, err := crCli.CrV1alpha1().RepositoryServers(rs.GetNamespace()).Get(ctx, rs.GetName(), metav1.GetOptions{})
		return repositoryServer.Status.Progress == crv1alpha1.Ready, err
	})
	if pollErr != nil {
		return errors.Wrapf(pollErr, "Failed while waiting for Repository Server %s/%s to be Ready", rs.GetNamespace(), rs.GetName())
	}
	return nil
}

func verifyObjectParams(p *PerformParams, cli kubernetes.Interface, osCli osversioned.Interface, ctx context.Context) error {
	var err error
	for _, obj := range p.Objects {
		switch obj.Kind {
		case param.DeploymentKind:
			_, err = cli.AppsV1().Deployments(obj.Namespace).Get(ctx, obj.Name, metav1.GetOptions{})
		case param.StatefulSetKind:
			_, err = cli.AppsV1().StatefulSets(obj.Namespace).Get(ctx, obj.Name, metav1.GetOptions{})
		case param.DeploymentConfigKind:
			// use open shift client to get the deployment config resource
			_, err = osCli.AppsV1().DeploymentConfigs(obj.Namespace).Get(ctx, obj.Name, metav1.GetOptions{})
		case param.PVCKind:
			_, err = cli.CoreV1().PersistentVolumeClaims(obj.Namespace).Get(ctx, obj.Name, metav1.GetOptions{})
		case param.NamespaceKind:
			_, err = cli.CoreV1().Namespaces().Get(ctx, obj.Name, metav1.GetOptions{})
		default:
			gvr := schema.GroupVersionResource{
				Group:    obj.Group,
				Version:  obj.APIVersion,
				Resource: obj.Resource,
			}
			_, err = kube.FetchUnstructuredObject(ctx, gvr, obj.Namespace, obj.Name)
		}
		if err != nil {
			return errors.Wrapf(err, "Please make sure '%s' with name '%s' exists in namespace '%s'", obj.Kind, obj.Name, obj.Namespace)
		}
	}
	return nil
}

func isDataMoverProvided(cmd *cobra.Command) error {
	profile := cmd.Flags().Lookup(profileFlagName).Value.String()
	repositoryServer := cmd.Flags().Lookup(repositoryServerFlagName).Value.String()
	if profile == "" && repositoryServer == "" {
		return errors.New("Neither --profile nor --repository-server flag is provided.\nAction might fail if blueprint is using these resources.")
	}
	return nil
}
