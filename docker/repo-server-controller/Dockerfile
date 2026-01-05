FROM registry.access.redhat.com/ubi9/ubi-minimal:9.7@sha256:6fc28bcb6776e387d7a35a2056d9d2b985dc4e26031e98a2bd35a7137cd6fd71

LABEL maintainer="Kanister maintainers<kanister.maintainers@veeam.com>"

RUN microdnf -y update openssl-libs

ADD repo-server-controller /repo-server-controller
ENTRYPOINT ["/repo-server-controller"]
