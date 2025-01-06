FROM registry.access.redhat.com/ubi9/ubi-minimal:9.5

LABEL maintainer="Kanister maintainers<kanister.maintainers@veeam.com>"

RUN microdnf -y update openssl-libs

ADD repo-server-controller /repo-server-controller
ENTRYPOINT ["/repo-server-controller"]
