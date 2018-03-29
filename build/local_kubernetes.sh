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
export KUBE_VERSION=${KUBE_VERSION:-v1.8.0}
declare -a REQUIRED_BINS=( iptables docker sudo jq )

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

start_minikube() {

    if ! command -v minikube
    then
        get_minikube
    fi

    minikube start --vm-driver=none --mount --kubernetes-version=${KUBE_VERSION}
    wait_for_minikube_nodes
    wait_for_pods
}

stop_minikube() {
   minikube stop
}

get_minikube() {
    check_or_get_dependencies
    mkdir $HOME/.kube || true
    touch $HOME/.kube/config
    curl -Lo minikube https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64 && chmod +x minikube
    ln -sf $(pwd)/minikube /usr/bin/minikube
}

wait_for_minikube_nodes() {
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
    local pod_status=$(kubectl get pod --namespace=${namespace} -o json | jq -j ".items | .[] | .status | .containerStatuses | .[] | .ready")
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
        if ! pod_status=$(kubectl get pod --namespace=${namespace} -o json | jq -j ".items | .[] | .status | .containerStatuses | .[] | .ready")
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
  get_minikube: installs minikube
  start_minikube : minikube start
  stop_minikube : minikube stop
EOM
    exit 1
}

[ ${#@} -gt 0 ] || usage
case "${1}" in
        # Alphabetically sorted
        get_minikube)
            time -p get_minikube
            ;;
        start_minikube)
            time -p start_minikube
            ;;
        stop_minikube)
            time -p stop_minikube
            ;;
        *)
            usage
            exit 1
esac
