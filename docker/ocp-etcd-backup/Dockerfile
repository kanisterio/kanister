# Keeper image for the OpenShift etcd backup blueprint.
#
# Contents:
#   - etcdctl / etcdutl : cluster-backup.sh calls etcdctl from PATH; etcdutl
#                         inspects/verifies snapshots.
#   - kubectl           : checks control-plane operators are not mid-rollout
#                         before backup, against the in-cluster API.
#   - bash, tar, gzip   : run cluster-backup.sh and archive its output.
#   - coreutils         : `sync` (flush page cache before the PVC snapshot), ls.
#
# cluster-backup.sh is not baked in; it is run from the control-plane node via
# the read-only /etc/kubernetes hostPath mount:
#   /etc/kubernetes/static-pod-resources/etcd-certs/configmaps/etcd-scripts/cluster-backup.sh

ARG ETCD_SOURCE_IMAGE=quay.io/coreos/etcd:v3.5.18@sha256:d0a641d5fbcc89678c931a61b7de7b8a1cf097149f135c9c73bc81d076a1494b
# kubectl is matched to the cluster Kubernetes version (OCP 4.18 = k8s 1.31).
ARG KUBECTL_VERSION=v1.31.6

FROM ${ETCD_SOURCE_IMAGE} AS etcd-src

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest@sha256:5ed244b62bbf4095080144d9d35eb8fcd3d39a9801f94aadd63b9d10978a01ae
ARG KUBECTL_VERSION
ARG TARGETARCH

RUN microdnf update -y && \
    microdnf install -y bash ca-certificates tar gzip coreutils-single curl-minimal && \
    microdnf clean all

RUN curl -fsSL "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/${TARGETARCH}/kubectl" \
      -o /usr/local/bin/kubectl && \
    chmod 0755 /usr/local/bin/kubectl

COPY --from=etcd-src /usr/local/bin/etcdctl /usr/local/bin/etcdctl
COPY --from=etcd-src /usr/local/bin/etcdutl /usr/local/bin/etcdutl

RUN /usr/local/bin/etcdctl version && /usr/local/bin/etcdutl version && \
    /usr/local/bin/kubectl version --client

# The keeper just needs to stay alive so the blueprint can KubeExec into it and
# so an admin can inspect the backup artifacts on the PVC.

ENTRYPOINT ["/bin/bash", "-c"]
CMD ["echo 'ocp-etcd-backup keeper ready'; sleep infinity"]
