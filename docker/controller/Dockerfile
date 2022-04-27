FROM registry.access.redhat.com/ubi8/ubi-minimal:8.5-243

LABEL maintainer="Tom Manville<tom@kasten.io>"

RUN microdnf update openssl-libs

ADD controller /controller
ENTRYPOINT ["/controller"]
