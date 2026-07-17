FROM registry.access.redhat.com/ubi9/ubi-minimal:9.8@sha256:062c52ff973065752b0965787649db2bcf551a6c727a00e95a3eb42cebadbdab

LABEL maintainer="Kanister maintainers<kanister.maintainers@veeam.com>"

RUN microdnf -y update openssl-libs

ADD repo-server-controller /repo-server-controller
ENTRYPOINT ["/repo-server-controller"]
