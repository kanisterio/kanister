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
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/consts"
	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
	"github.com/kanisterio/kanister/pkg/progress"
	"github.com/kanisterio/kanister/pkg/restic"
)

const (
	// BackupDataAllFuncName gives the name of the function
	BackupDataAllFuncName = "BackupDataAll"
	// BackupDataAllNamespaceArg provides the namespace
	BackupDataAllNamespaceArg = "namespace"
	// BackupDataAllPodsArg provides the pods connected to the data volumes
	BackupDataAllPodsArg = "pods"
	// BackupDataAllContainerArg provides the container on which the backup is taken
	BackupDataAllContainerArg = "container"
	// BackupDataAllIncludePathArg provides the path of the volume or sub-path for required backup
	BackupDataAllIncludePathArg = "includePath"
	// BackupDataAllBackupArtifactPrefixArg provides the path to store artifacts on the object store
	BackupDataAllBackupArtifactPrefixArg = "backupArtifactPrefix"
	// BackupDataAllEncryptionKeyArg provides the encryption key to be used for backups
	BackupDataAllEncryptionKeyArg = "encryptionKey"
	// BackupDataAllOutput is the key name of the output generated by BackupDataAll func
	BackupDataAllOutput = "BackupAllInfo"
	// InsecureTLS is the key name which provides an option to a user to disable tls
	InsecureTLS = "insecure-tls"
)

type BackupInfo struct {
	PodName   string
	BackupID  string
	BackupTag string
}

func init() {
	_ = kanister.Register(&backupDataAllFunc{})
}

var _ kanister.Func = (*backupDataAllFunc)(nil)

type backupDataAllFunc struct {
	progressPercent string
}

func (*backupDataAllFunc) Name() string {
	return BackupDataAllFuncName
}

func (b *backupDataAllFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	// Set progress percent
	b.progressPercent = progress.StartedPercent
	defer func() { b.progressPercent = progress.CompletedPercent }()

	var namespace, pods, container, includePath, backupArtifactPrefix, encryptionKey string
	var err error
	var insecureTLS bool
	if err = Arg(args, BackupDataAllNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, BackupDataAllContainerArg, &container); err != nil {
		return nil, err
	}
	if err = Arg(args, BackupDataAllIncludePathArg, &includePath); err != nil {
		return nil, err
	}
	if err = Arg(args, BackupDataAllBackupArtifactPrefixArg, &backupArtifactPrefix); err != nil {
		return nil, err
	}
	if err = OptArg(args, BackupDataAllPodsArg, &pods, ""); err != nil {
		return nil, err
	}
	if err = OptArg(args, BackupDataAllEncryptionKeyArg, &encryptionKey, restic.GeneratePassword()); err != nil {
		return nil, err
	}
	if err = OptArg(args, InsecureTLS, &insecureTLS, false); err != nil {
		return nil, err
	}

	if err = ValidateProfile(tp.Profile); err != nil {
		return nil, errors.Wrapf(err, "Failed to validate Profile")
	}

	backupArtifactPrefix = ResolveArtifactPrefix(backupArtifactPrefix, tp.Profile)

	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	var ps []string
	if pods == "" {
		switch {
		case tp.Deployment != nil:
			ps = tp.Deployment.Pods
		case tp.StatefulSet != nil:
			ps = tp.StatefulSet.Pods
		default:
			return nil, errors.New("Failed to get pods")
		}
	} else {
		ps = strings.Fields(pods)
	}
	ctx = field.Context(ctx, consts.ContainerNameKey, container)
	return backupDataAll(ctx, cli, namespace, ps, container, backupArtifactPrefix, includePath, encryptionKey, insecureTLS, tp)
}

func (*backupDataAllFunc) RequiredArgs() []string {
	return []string{
		BackupDataAllNamespaceArg,
		BackupDataAllContainerArg,
		BackupDataAllIncludePathArg,
		BackupDataAllBackupArtifactPrefixArg,
	}
}

func (*backupDataAllFunc) Arguments() []string {
	return []string{
		BackupDataAllNamespaceArg,
		BackupDataAllContainerArg,
		BackupDataAllIncludePathArg,
		BackupDataAllBackupArtifactPrefixArg,
		BackupDataAllPodsArg,
		BackupDataAllEncryptionKeyArg,
		InsecureTLS,
	}
}

func backupDataAll(ctx context.Context, cli kubernetes.Interface, namespace string, ps []string, container string, backupArtifactPrefix, includePath, encryptionKey string,
	insecureTLS bool, tp param.TemplateParams) (map[string]interface{}, error) {
	errChan := make(chan error, len(ps))
	outChan := make(chan BackupInfo, len(ps))
	Output := make(map[string]BackupInfo)
	// Run the command
	for _, pod := range ps {
		go func(pod string, container string) {
			ctx = field.Context(ctx, consts.PodNameKey, pod)
			backupOutputs, err := backupData(ctx, cli, namespace, pod, container, fmt.Sprintf("%s/%s", backupArtifactPrefix, pod), includePath, encryptionKey, insecureTLS, tp)
			errChan <- errors.Wrapf(err, "Failed to backup data for pod %s", pod)
			outChan <- BackupInfo{PodName: pod, BackupID: backupOutputs.backupID, BackupTag: backupOutputs.backupTag}
		}(pod, container)
	}
	errs := make([]string, 0, len(ps))
	for i := 0; i < len(ps); i++ {
		err := <-errChan
		output := <-outChan
		if err != nil {
			errs = append(errs, err.Error())
		} else {
			Output[output.PodName] = output
		}
	}
	if len(errs) != 0 {
		return nil, errors.New(strings.Join(errs, "\n"))
	}
	manifestData, err := json.Marshal(Output)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to encode JSON data")
	}
	return map[string]interface{}{
		BackupDataAllOutput:   string(manifestData),
		FunctionOutputVersion: kanister.DefaultVersion,
	}, nil
}

func (b *backupDataAllFunc) ExecutionProgress() (crv1alpha1.PhaseProgress, error) {
	metav1Time := metav1.NewTime(time.Now())
	return crv1alpha1.PhaseProgress{
		ProgressPercent:    b.progressPercent,
		LastTransitionTime: &metav1Time,
	}, nil
}
