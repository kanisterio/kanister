#! /bin/bash

# generate_secrets.bash
#
# This source file is part of the FoundationDB open source project
#
# Copyright 2018-2019 Apple Inc. and the FoundationDB project authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

# This script generates secrets with test certs for use in local testing.

kubectl delete secrets -l app=fdb-kubernetes-operator
kubectl create secret generic fdb-kubernetes-operator-secrets --from-file=config/test-certs/key.pem --from-file=config/test-certs/cert.pem
kubectl patch secret fdb-kubernetes-operator-secrets --type='json' -p='[{"op": "add", "path": "/metadata/labels", "value":{"app":"fdb-kubernetes-operator"}}]'