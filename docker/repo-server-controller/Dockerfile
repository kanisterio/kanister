FROM registry.access.redhat.com/ubi9/ubi-minimal:9.8@sha256:457cb288730d29e2a59823b3b56341466fa94f3158e399c5ab2c4409fee0c562

LABEL maintainer="Kanister maintainers<kanister.maintainers@veeam.com>"

RUN microdnf -y update openssl-libs

ADD repo-server-controller /repo-server-controller
ENTRYPOINT ["/repo-server-controller"]
