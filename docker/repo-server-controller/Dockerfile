FROM registry.access.redhat.com/ubi9/ubi-minimal:9.7@sha256:bb08f2300cb8d12a7eb91dddf28ea63692b3ec99e7f0fa71a1b300f2756ea829

LABEL maintainer="Kanister maintainers<kanister.maintainers@veeam.com>"

RUN microdnf -y update openssl-libs

ADD repo-server-controller /repo-server-controller
ENTRYPOINT ["/repo-server-controller"]
