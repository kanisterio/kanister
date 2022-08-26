FROM registry.access.redhat.com/ubi8/ubi-minimal:-

LABEL maintainer="Tom Manville<tom@kasten.io>"

RUN microdnf update openssl-libs

ADD controller /controller
ENTRYPOINT ["/controller"]
