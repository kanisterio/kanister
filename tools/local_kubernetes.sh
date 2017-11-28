#!/usr/bin/env bash
#
# The wrapper for https://kubernetes-v1-4.github.io/docs/getting-started-guides/locally/#requirements
# also script checks if shippable cache is used for any thing

set -o errexit
set -o nounset
set -o xtrace
set -o pipefail

readonly BASE_DIR=$(dirname ${0})
readonly K8S_COMPILE_TIME=15m
readonly HACK_K8S_CONFIG=/var/run/kubernetes/admin.kubeconfig
export MINIKUBE_WANTUPDATENOTIFICATION=false
export MINIKUBE_WANTREPORTERRORPROMPT=false
export MINIKUBE_HOME=$HOME
export CHANGE_MINIKUBE_NONE_USER=true 

export KUBECONFIG=$HOME/.kube/config

start_minikube() {
    
    if ! command -v minikube 
    then
        get_minikube
    fi

    if ! command -v iptables
    then
        apt-get install -y iptables
    fi

    ./minikube start --vm-driver=none --mount --kubernetes-version=v1.7.5
    wait_for_all
}

stop_minikube() {
   ./minikube stop
}

get_minikube() {
    mkdir $HOME/.kube || true
    touch $HOME/.kube/config
    curl -Lo minikube https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64 && chmod +x minikube
    curl -Lo kubectl https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl && chmod +x kubectl
    ln -sf $(pwd)/minikube /usr/bin/minikube
}

wait_for_all() {
    local all_status=$(kubectl get all --namespace=kube-system -o json | jq -j ".items | .[] | .status | .containerStatuses | .[] | .ready")
    local retries=10
    while [[  ${all_status} == *false* ]]
    do
        if [[ ${retries} -le 0 ]]
        then
            echo "Error some objects are not ready"
            kubectl get all --namespace=kube-system
            return 1
        fi
        sleep 10
        all_status=$(kubectl get all --namespace=kube-system -o json | jq -j ".items | .[] | .status | .containerStatuses | .[] | .ready")
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
