FROM registry.access.redhat.com/ubi8/ubi-minimal:8.6-902.1661794353

LABEL maintainer="Tom Manville<tom@kasten.io>"

RUN microdnf update openssl-libs

ADD controller /controller
ENTRYPOINT ["/controller"]
