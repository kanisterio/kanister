FROM registry.access.redhat.com/ubi9/ubi-minimal:9.8@sha256:8eb2830d0936237fc13a1f2f7e45aecf90d69043380ad167fad0343632937f41

LABEL maintainer="Kanister maintainers<kanister.maintainers@veeam.com>"

RUN microdnf -y update openssl-libs

ADD controller /controller
ENTRYPOINT ["/controller"]
