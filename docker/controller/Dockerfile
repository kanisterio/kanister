FROM registry.access.redhat.com/ubi9/ubi-minimal:9.8@sha256:2e8edce823a48e51858f1fad3ff4cbf6875ce8a3f86b9eecf298bc2050c8652a

LABEL maintainer="Kanister maintainers<kanister.maintainers@veeam.com>"

RUN microdnf -y update openssl-libs

ADD controller /controller
ENTRYPOINT ["/controller"]
