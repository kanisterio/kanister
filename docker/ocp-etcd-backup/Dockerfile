# Keeper image for the OpenShift etcd backup blueprint.
#
# The image needs:
#   - etcdctl / etcdutl    : the host script cluster-backup.sh calls `etcdctl`
#                            from PATH (it skips its own download when etcdctl is
#                            already present), and etcdutl is used to inspect /
#                            verify snapshots.
#   - kubectl              : the backupPrehook checks that the control-plane
#                            operators are not mid-rollout before backing up
#                            (`kubectl get co ... Progressing`). cluster-backup.sh
#                            performs this check itself only WITHOUT --force, using
#                            `oc` + the node localhost.kubeconfig (server
#                            https://localhost:6443) which a non-hostNetwork pod
#                            cannot reach. So we run --force and replicate the
#                            check against the in-cluster API — exactly as the
#                            upstream etcd-bp-azcopy blueprint does.
#   - bash, tar, gzip      : run the host cluster-backup.sh and archive output.
#   - coreutils/util-linux : `sync` (flush page cache before the PVC snapshot),
#                            ls, etc.
#
# cluster-backup.sh itself is NOT baked in — it is invoked from the control-plane
# node via the read-only /etc/kubernetes hostPath mount:
#   /etc/kubernetes/static-pod-resources/etcd-certs/configmaps/etcd-scripts/cluster-backup.sh

ARG ETCD_SOURCE_IMAGE=quay.io/coreos/etcd:v3.5.18
# kubectl is matched to the cluster Kubernetes version (OCP 4.18 = k8s 1.31).
ARG KUBECTL_VERSION=v1.31.6

FROM ${ETCD_SOURCE_IMAGE} AS etcd-src

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest
ARG KUBECTL_VERSION

RUN microdnf update -y && \
    microdnf install -y bash ca-certificates tar gzip coreutils-single util-linux curl-minimal && \
    microdnf clean all

RUN curl -fsSL "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl" \
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