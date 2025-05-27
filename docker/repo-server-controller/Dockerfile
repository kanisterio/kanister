FROM registry.access.redhat.com/ubi9/ubi-minimal:9.6@sha256:92b1d5747a93608b6adb64dfd54515c3c5a360802db4706765ff3d8470df6290

LABEL maintainer="Kanister maintainers<kanister.maintainers@veeam.com>"

RUN microdnf -y update openssl-libs

ADD repo-server-controller /repo-server-controller
ENTRYPOINT ["/repo-server-controller"]
