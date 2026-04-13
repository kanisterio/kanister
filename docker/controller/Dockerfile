FROM registry.access.redhat.com/ubi9/ubi-minimal:9.7@sha256:83006d535923fcf1345067873524a3980316f51794f01d8655be55d6e9387183

LABEL maintainer="Kanister maintainers<kanister.maintainers@veeam.com>"

RUN microdnf -y update openssl-libs

ADD controller /controller
ENTRYPOINT ["/controller"]
