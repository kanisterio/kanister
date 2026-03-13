FROM registry.access.redhat.com/ubi9/ubi-minimal:9.7@sha256:c7d44146f826037f6873d99da479299b889473492d3c1ab8af86f08af04ec8a0

LABEL maintainer="Kanister maintainers<kanister.maintainers@veeam.com>"

RUN microdnf -y update openssl-libs

ADD controller /controller
ENTRYPOINT ["/controller"]
