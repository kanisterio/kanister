FROM registry.access.redhat.com/ubi9/ubi-minimal:9.1.0-1760.1675784957

LABEL maintainer="Tom Manville<tom@kasten.io>"

RUN microdnf -y update openssl-libs

ADD controller /controller
ENTRYPOINT ["/controller"]
