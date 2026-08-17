FROM registry.access.redhat.com/ubi9/ubi-minimal:9.8@sha256:7c372902c8d211db2d25c8277ba534a73b92742a334874dced829a63b0f21221

LABEL maintainer="Kanister maintainers<kanister.maintainers@veeam.com>"

RUN microdnf -y update openssl-libs

ADD controller /controller
ENTRYPOINT ["/controller"]
