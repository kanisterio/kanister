FROM registry.access.redhat.com/ubi9/ubi-minimal:9.7@sha256:90bd85dcd061d1ad6dbda70a867c41958c04a86462d05c631f8205e8870f28f8

LABEL maintainer="Kanister maintainers<kanister.maintainers@veeam.com>"

RUN microdnf -y update openssl-libs

ADD repo-server-controller /repo-server-controller
ENTRYPOINT ["/repo-server-controller"]
