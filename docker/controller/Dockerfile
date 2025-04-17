FROM registry.access.redhat.com/ubi9/ubi-minimal:9.5@sha256:ac61c96b93894b9169221e87718733354dd3765dd4a62b275893c7ff0d876869

LABEL maintainer="Kanister maintainers<kanister.maintainers@veeam.com>"

RUN microdnf -y update openssl-libs

ADD controller /controller
ENTRYPOINT ["/controller"]
