FROM registry.access.redhat.com/ubi9/ubi-minimal:9.6@sha256:f172b3082a3d1bbe789a1057f03883c1113243564f01cd3020e27548b911d3f8

LABEL maintainer="Kanister maintainers<kanister.maintainers@veeam.com>"

RUN microdnf -y update openssl-libs

ADD repo-server-controller /repo-server-controller
ENTRYPOINT ["/repo-server-controller"]
