FROM registry.access.redhat.com/ubi9/ubi-minimal:9.7@sha256:175bafd5bc7893540ed6234bb979acfe3574fd6570e6762bbc527c757f854cea

LABEL maintainer="Kanister maintainers<kanister.maintainers@veeam.com>"

RUN microdnf -y update openssl-libs

ADD controller /controller
ENTRYPOINT ["/controller"]
