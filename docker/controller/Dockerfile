FROM registry.access.redhat.com/ubi8/ubi-minimal:8.7-923.1669829893

LABEL maintainer="Tom Manville<tom@kasten.io>"

RUN microdnf update openssl-libs

ADD controller /controller
ENTRYPOINT ["/controller"]
