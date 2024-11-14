FROM registry.access.redhat.com/ubi9/ubi-minimal:9.5-1731518200

LABEL maintainer="Tom Manville<tom@kasten.io>"

RUN microdnf -y update openssl-libs

ADD controller /controller
ENTRYPOINT ["/controller"]
