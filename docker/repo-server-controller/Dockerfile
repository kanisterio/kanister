FROM registry.access.redhat.com/ubi9/ubi-minimal:9.7@sha256:fe688da81a696387ca53a4c19231e99289591f990c904ef913c51b6e87d4e4df

LABEL maintainer="Kanister maintainers<kanister.maintainers@veeam.com>"

RUN microdnf -y update openssl-libs

ADD repo-server-controller /repo-server-controller
ENTRYPOINT ["/repo-server-controller"]
