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
hack_cluster_up_cmd="hack/local-up-cluster.sh"
WAIT_FOR_KUBE="while [ ! -f ${HACK_K8S_CONFIG} ];do sleep 1;done"

start_local_kube() {
    if [[ ! -d ${BASE_DIR}/../kubernetes ]]
    then
        git clone https://github.com/kubernetes/kubernetes.git
    fi

    if [[ -d ${BASE_DIR}/../kubernetes/_output ]]
    then
        hack_cluster_up_cmd="${hack_cluster_up_cmd} -O"
    fi

    if ! command -v etcd 
    then
        get_etcd
    fi

    pushd ${BASE_DIR}/../kubernetes
    export ENABLE_DAEMON=true 
    output=$(${hack_cluster_up_cmd} &)
    popd
    timeout ${K8S_COMPILE_TIME} bash -c "$WAIT_FOR_KUBE" 
    cp ${HACK_K8S_CONFIG} ~/.kube/config
    wait_for_pods
    echo ${output} 
}

stop_local_kube() {
   #kill $(ps -ef | grep ${hack_cluster_up_cmd} | grep -v grep | awk '{print $2}') 
   rm -f ${HACK_K8S_CONFIG}
   killall hyperkube
   killall etcd
}

get_etcd() {
    #all credits to https://github.com/coreos/etcd/releases/

    ETCD_VER=v3.2.10
    GOOGLE_URL=https://storage.googleapis.com/etcd
    GITHUB_URL=https://github.com/coreos/etcd/releases/download
    DOWNLOAD_URL=${GOOGLE_URL}
    rm -f /tmp/etcd-${ETCD_VER}-linux-amd64.tar.gz
    rm -rf /tmp/etcd-download-test && mkdir -p /tmp/etcd-download-test
    curl -L ${DOWNLOAD_URL}/${ETCD_VER}/etcd-${ETCD_VER}-linux-amd64.tar.gz -o /tmp/etcd-${ETCD_VER}-linux-amd64.tar.gz
    tar xzvf /tmp/etcd-${ETCD_VER}-linux-amd64.tar.gz -C /tmp/etcd-download-test --strip-components=1
    rm -f /tmp/etcd-${ETCD_VER}-linux-amd64.tar.gz
    ln -s /tmp/etcd-download-test/etcd /usr/bin/etcd
    etcd --version
}

wait_for_pods() {
    local pods_status=$(kubectl get pods --namespace=kube-system -o json | jq -j ".items | .[] | .status | .containerStatuses | .[] | .ready")
    local retries=10
    while [[  ${pods_status} == *false* ]]
    do
        if [[ ${retries} -le 0 ]]
        then
            echo "Error some pods are not ready"
            kubectl get pods --namespace=kube-system
            return 1
        fi
        sleep 10
        pods_status=$(kubectl get pods --namespace=kube-system -o json | jq -j ".items | .[] | .status | .containerStatuses | .[] | .ready")
        retries=$((retries-1))
    done
    kubectl cluster-info
}


usage() {
    cat <<EOM
Usage: ${0} <operation> 
Where operation is one of the following:
  get_etcd : Intalls etcd v3.2.10
  start_local_kube : Check for required foolders. Intalls etcd, clones k8s repo, execute ${hack_cluster_up_cmd} 
  stop_local_kube : Kills pid for ${hack_cluster_up_cmd}
EOM
    exit 1
}

[ ${#@} -gt 0 ] || usage
case "${1}" in
        # Alphabetically sorted
        get_etcd)
            time -p get_etcd
            ;;
        start_local_kube)
            time -p start_local_kube
            ;;
        stop_local_kube)
            time -p stop_local_kube
            ;;
        *)
            usage
            exit 1    
esac
