FROM registry.access.redhat.com/ubi9/ubi-minimal:9.7@sha256:7d4e47500f28ac3a2bff06c25eff9127ff21048538ae03ce240d57cf756acd00

LABEL maintainer="Kanister maintainers<kanister.maintainers@veeam.com>"

RUN microdnf -y update openssl-libs

ADD controller /controller
ENTRYPOINT ["/controller"]
