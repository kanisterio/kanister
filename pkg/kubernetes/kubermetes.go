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
package kubernetes

import (
	"context"
)

// TODO make a generic function
type KubeClient interface {
	// similar to oc new-app ...
	InstallOperator(ctx context.Context, namespace, yamlFileRepo, strimziYaml string) (string, error)
	InstallKafka(ctx context.Context, namespace, yamlFileRepo, kafkaConfigPath string) (string, error)
	// DeleteApp delete an app from the kubernetes cluster
	// similar to oc delete all -n <ns> -l <label>
	CreateConfigMap(ctx context.Context, configMapName, namespace, yamlFileRepo, sinkConfigPath, sourceConfigPath, kafkaConfigPath string) (string, error)
	DeleteConfigMap(ctx context.Context, namespace, configMapName string) (string, error)
	DeleteKafka(ctx context.Context, namespace, yamlFileRepo, kafkaConfigPath string) (string, error)
	DeleteOperator(ctx context.Context, namespace, yamlFileRepo, strimziYaml string) (string, error)
	Ping(ctx context.Context, namespace string) (string, error)
	Insert(ctx context.Context, namespace string) (string, error)
}
