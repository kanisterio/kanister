#!/usr/bin/env bash
#
# The wrapper for https://kubernetes-v1-4.github.io/docs/getting-started-guides/locally/#requirements
# also script checks if shippable cache is used for any thing

set -o errexit
set -o nounset
set -o xtrace
set -o pipefail

readonly BASE_DIR=$(dirname ${0})
export MINIKUBE_WANTUPDATENOTIFICATION=false
export MINIKUBE_WANTREPORTERRORPROMPT=false
export MINIKUBE_HOME=$HOME
export CHANGE_MINIKUBE_NONE_USER=true
export KUBECONFIG=$HOME/.kube/config
export KUBE_VERSION=${KUBE_VERSION:-"v1.13.7"}
export KIND_VERSION=${KIND_VERSION:-"v0.4.0"}
export LOCAL_CLUSTER_NAME=${LOCAL_CLUSTER_NAME:-"kanister"}
declare -a REQUIRED_BINS=( docker jq go )

if command -v apt-get
then
    lin_repo_pre_cmd="apt-get install -y "
elif command -v apk
then
    lin_repo_pre_cmd="apk add --update "
else
    echo "apk or apt-get is supported at this moment"
    exit 1
fi

check_or_get_dependencies() {
    for dep in ${REQUIRED_BINS[@]}
    do
        if ! command -v ${dep}
        then
            echo "Missing ${dep}. Trying to install"
            if ! err=$(${lin_repo_pre_cmd} ${dep} 2>&1)
            then
                echo "Insatlletion failed with $err"
                exit 1
            fi
        fi
    done
}

start_localkube() {
    if ! command -v kind
    then
        get_localkube
    fi
    kind create cluster --name ${LOCAL_CLUSTER_NAME} --image=kindest/node:${KUBE_VERSION}
    if [ -e ${KUBECONFIG} ]; then
        cp -fr ${KUBECONFIG} ${HOME}/.kube/config_bk
    fi
    KUBECONFIG=$(kind get kubeconfig-path --name="kanister")
    export KUBECONFIG=${KUBECONFIG}:${HOME}/.kube/config_bk; kubectl config view --flatten > "${HOME}/.kube/config"
    wait_for_nodes
    wait_for_pods
}

stop_localkube() {
    if ! command -v kind
    then
        get_localkube
    fi
    kind delete cluster --name ${LOCAL_CLUSTER_NAME}
}

get_localkube() {
    mkdir $HOME/.kube || true
    touch $HOME/.kube/config
    GO111MODULE="on" go get sigs.k8s.io/kind@${KIND_VERSION}
}

wait_for_nodes() {
    local nodes_ready=$(kubectl get nodes 2>/dev/null | grep -c Ready)
    local retries=20
    while [[  ${nodes_ready} -eq 0 ]]
    do
        if [[ ${retries} -le 0 ]]
        then
            echo "Minikube nodes are not ready"
            kubectl get nodes
            minikube status
            return 1
        fi
        sleep 5
        if ! nodes_ready=$(kubectl get nodes 2>/dev/null | grep -c Ready)
        then
            nodes_ready=0
        fi
        retries=$((retries-1))
    done
    kubectl get nodes
}

wait_for_pods() {
    local namespace=${1:-"kube-system"}
    local pod_status=$(kubectl get pod --namespace=${namespace} -o json | jq -j '.items | .[] | .status | .containerStatuses | .[]? | .state.running != null or .state.terminated.reason == "Completed"')
    local retries=20

    while [[  ${pod_status} == *false* ]] || [[ ${pod_status} == '' ]]
    do
        if [[ ${retries} -le 0 ]]
        then
            echo "Error some objects are not ready"
            kubectl get pod --namespace=${namespace}
            return 1
        fi
        sleep 5
        if ! pod_status=$(kubectl get pod --namespace=${namespace} -o json | jq -j '.items | .[] | .status | .containerStatuses | .[]? | .state.running != null or .state.terminated.reason == "Completed"')
        then
             pod_status=''
        fi
        retries=$((retries-1))
    done
    kubectl cluster-info
}

usage() {
    cat <<EOM
Usage: ${0} <operation>
Where operation is one of the following:
  get_localkube: installs kind
  start_localkube : localkube start
  stop_localkube : localkube stop
EOM
    exit 1
}

[ ${#@} -gt 0 ] || usage
check_or_get_dependencies
case "${1}" in
        # Alphabetically sorted
        get_localkube)
            time -p get_localkube
            ;;
        start_localkube)
            time -p start_localkube
            ;;
        stop_localkube)
            time -p stop_localkube
            ;;
        *)
            usage
            exit 1
esac
