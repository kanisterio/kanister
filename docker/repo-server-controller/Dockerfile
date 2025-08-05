FROM registry.access.redhat.com/ubi9/ubi-minimal:9.6@sha256:aaf57d07d6e192bcb72190f9841f41501b8b3dffdc5e60db2f7a668c2513ab6b

LABEL maintainer="Kanister maintainers<kanister.maintainers@veeam.com>"

RUN microdnf -y update openssl-libs

ADD repo-server-controller /repo-server-controller
ENTRYPOINT ["/repo-server-controller"]
