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

package function

import (
	"context"
	"strconv"
	"strings"
	"time"

	osversioned "github.com/openshift/client-go/apps/clientset/versioned"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/utils"
)

const (
	// ScaleWorkloadFuncName gives the function name
	ScaleWorkloadFuncName     = "ScaleWorkload"
	ScaleWorkloadNamespaceArg = "namespace"
	ScaleWorkloadNameArg      = "name"
	ScaleWorkloadKindArg      = "kind"
	ScaleWorkloadReplicas     = "replicas"
	ScaleWorkloadWaitArg      = "waitForReady"

	outputArtifactOriginalReplicaCount = "originalReplicaCount"
)

func init() {
	_ = kanister.Register(&scaleWorkloadFunc{})
}

var (
	_ kanister.Func = (*scaleWorkloadFunc)(nil)
)

type scaleWorkloadFunc struct {
	progressPercent string
	namespace       string
	kind            string
	name            string
	replicas        int32
	waitForReady    bool
}

func (*scaleWorkloadFunc) Name() string {
	return ScaleWorkloadFuncName
}

func (s *scaleWorkloadFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	// Set progress percent
	s.progressPercent = progress.StartedPercent
	defer func() { s.progressPercent = progress.CompletedPercent }()

	err := s.setArgs(tp, args)
	if err != nil {
		return nil, err
	}

	cfg, err := kube.LoadConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to load Kubernetes config")
	}
	cli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	switch strings.ToLower(s.kind) {
	case param.StatefulSetKind:
		count, err := kube.StatefulSetReplicas(ctx, cli, s.namespace, s.name)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			outputArtifactOriginalReplicaCount: count,
		}, kube.ScaleStatefulSet(ctx, cli, s.namespace, s.name, s.replicas, s.waitForReady)
	case param.DeploymentKind:
		count, err := kube.DeploymentReplicas(ctx, cli, s.namespace, s.name)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			outputArtifactOriginalReplicaCount: count,
		}, kube.ScaleDeployment(ctx, cli, s.namespace, s.name, s.replicas, s.waitForReady)
	case param.DeploymentConfigKind:
		osCli, err := osversioned.NewForConfig(cfg)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to create OpenShift client")
		}
		count, err := kube.DeploymentConfigReplicas(ctx, osCli, s.namespace, s.name)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			outputArtifactOriginalReplicaCount: count,
		}, kube.ScaleDeploymentConfig(ctx, cli, osCli, s.namespace, s.name, s.replicas, s.waitForReady)
	}
	return nil, errors.New("Workload type not supported " + s.kind)
}

func (*scaleWorkloadFunc) RequiredArgs() []string {
	return []string{ScaleWorkloadReplicas}
}

func (*scaleWorkloadFunc) Arguments() []string {
	return []string{
		ScaleWorkloadReplicas,
		ScaleWorkloadNamespaceArg,
		ScaleWorkloadNameArg,
		ScaleWorkloadKindArg,
		ScaleWorkloadWaitArg,
	}
}

func (r *scaleWorkloadFunc) Validate(args map[string]any) error {
	if err := utils.CheckSupportedArgs(r.Arguments(), args); err != nil {
		return err
	}

	return utils.CheckRequiredArgs(r.RequiredArgs(), args)
}

func (s *scaleWorkloadFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    s.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}

func (s *scaleWorkloadFunc) setArgs(tp param.TemplateParams, args map[string]interface{}) error {
	var rep interface{}
	waitForReady := true
	err := Arg(args, ScaleWorkloadReplicas, &rep)
	if err != nil {
		return err
	}

	var namespace, kind, name string
	var replicas int32

	switch val := rep.(type) {
	case int:
		replicas = int32(val)
	case int32:
		replicas = val
	case int64:
		replicas = int32(val)
	case string:
		var v int
		if v, err = strconv.Atoi(val); err != nil {
			return errors.Wrapf(err, "Cannot convert %s to int ", val)
		}
		replicas = int32(v)
	default:
		return errors.Errorf("Invalid arg type %T for Arg %s ", rep, ScaleWorkloadReplicas)
	}
	// Populate default values for optional arguments from template parameters
	switch {
	case tp.StatefulSet != nil:
		kind = param.StatefulSetKind
		name = tp.StatefulSet.Name
		namespace = tp.StatefulSet.Namespace
	case tp.Deployment != nil:
		kind = param.DeploymentKind
		name = tp.Deployment.Name
		namespace = tp.Deployment.Namespace
	case tp.DeploymentConfig != nil:
		kind = param.DeploymentConfigKind
		name = tp.DeploymentConfig.Name
		namespace = tp.DeploymentConfig.Namespace
	default:
		if !ArgExists(args, ScaleWorkloadNamespaceArg) || !ArgExists(args, ScaleWorkloadNameArg) || !ArgExists(args, ScaleWorkloadKindArg) {
			return errors.New("Workload information not available via defaults or namespace/name/kind parameters")
		}
	}

	err = OptArg(args, ScaleWorkloadNamespaceArg, &namespace, namespace)
	if err != nil {
		return err
	}
	err = OptArg(args, ScaleWorkloadNameArg, &name, name)
	if err != nil {
		return err
	}
	err = OptArg(args, ScaleWorkloadKindArg, &kind, kind)
	if err != nil {
		return err
	}
	err = OptArg(args, ScaleWorkloadWaitArg, &waitForReady, waitForReady)
	if err != nil {
		return err
	}
	s.kind = kind
	s.name = name
	s.namespace = namespace
	s.replicas = replicas
	s.waitForReady = waitForReady
	return nil
}
