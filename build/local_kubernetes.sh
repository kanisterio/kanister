#!/usr/bin/env bash
#
# Copyright 2019 The Kanister Authors.
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
export KUBE_VERSION=${KUBE_VERSION:-"v1.26.0"}
export KIND_VERSION=${KIND_VERSION:-"v0.18.0"}
export LOCAL_CLUSTER_NAME=${LOCAL_CLUSTER_NAME:-"kanister"}
export LOCAL_PATH_PROV_VERSION="v0.0.11"
export SNAPSHOTTER_VERSION="v6.2.1"
export HOSTPATH_DRIVER_VERSION="v1.12.1"
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

# This function is not used to create Kubernetes cluster in CI anymore. We are
# using `helm/kind-action@v1.4.0` instead in our github actions source file
# `.github/workflows/main.yaml` to create the cluster
start_localkube() {
    if ! command -v kind
    then
        get_localkube
    fi
    kind create cluster --name ${LOCAL_CLUSTER_NAME} --image=kindest/node:${KUBE_VERSION} -v 1
    if [ -e ${KUBECONFIG} ]; then
        cp -fr ${KUBECONFIG} ${HOME}/.kube/config_bk
    fi
    kind get kubeconfig --name="kanister" > "${HOME}/.kube/config"
    wait_for_nodes
    wait_for_pods
}

install_csi_hostpath_driver() {
    # Install VolumeSnapshot CRDs
    kubectl apply -fhttps://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/${SNAPSHOTTER_VERSION}/client/config/crd/snapshot.storage.k8s.io_{volumesnapshots.yaml,volumesnapshotclasses.yaml,volumesnapshotcontents.yaml}

    # Create snapshot controller
    kubectl apply -fhttps://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/${SNAPSHOTTER_VERSION}/deploy/kubernetes/snapshot-controller/{rbac-snapshot-controller.yaml,setup-snapshot-controller.yaml}

    # Deploy the CSI Hostpath Driver
    pushd /tmp
      git clone https://github.com/kubernetes-csi/csi-driver-host-path.git
      pushd csi-driver-host-path
        git checkout ${HOSTPATH_DRIVER_VERSION}
        sed -i 's/mountPropagation: Bidirectional/\#mountPropagation: Bidirectional/g' deploy/kubernetes-latest/hostpath/csi-hostpath-plugin.yaml

        ./deploy/kubernetes-latest/deploy.sh

        # Create StorageClass
        kubectl apply -f ./examples/csi-storageclass.yaml
      popd
    popd
}

check_csi_hostpath_driver_installed() {
    # Check VolumeSnapshot CRDs are installed
    if ! kubectl diff -fhttps://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/${SNAPSHOTTER_VERSION}/client/config/crd/snapshot.storage.k8s.io_{volumesnapshots.yaml,volumesnapshotclasses.yaml,volumesnapshotcontents.yaml} 2>&1 > /dev/null ; then
        echo "VolumeSnapshot CRDs are not installed."
        exit 1
    fi

    # Check snapshot controller created
    if ! kubectl diff -fhttps://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/${SNAPSHOTTER_VERSION}/deploy/kubernetes/snapshot-controller/{rbac-snapshot-controller.yaml,setup-snapshot-controller.yaml} 2>&1 > /dev/null ; then
        echo "Snapshot controller is not created."
        exit 1
    fi

    # Deploy the CSI Hostpath Driver
    pushd /tmp
      git clone https://github.com/kubernetes-csi/csi-driver-host-path.git
      pushd csi-driver-host-path
        # Check StorageClass created
        if ! kubectl diff -f ./examples/csi-storageclass.yaml 2>&1 > /dev/null ; then
            echo "StorageClass is not created."
            exit 1
        fi
      popd
    popd
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
  install_csi_hostpath_driver : installs CSI hostpath driver
EOM
    exit 1
}

[ ${#@} -gt 0 ] || usage
check_or_get_dependencies
case "${1}" in
        # Alphabetically sorted
        check_csi_hostpath_driver_installed)
            time -p check_csi_hostpath_driver_installed
            ;;
        get_localkube)
            time -p get_localkube
            ;;
        install_csi_hostpath_driver)
            time -p install_csi_hostpath_driver
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
