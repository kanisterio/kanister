FROM registry.access.redhat.com/ubi9/ubi-minimal:9.9@sha256:08d4c70f1baf6537a0728656cf7942d561e74a35916509d205b6519861266352

LABEL maintainer="Kanister maintainers<kanister.maintainers@veeam.com>"

RUN microdnf -y update openssl-libs

ADD controller /controller
ENTRYPOINT ["/controller"]
