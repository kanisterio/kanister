FROM registry.access.redhat.com/ubi9/ubi-minimal:9.4-1227.1726694542

LABEL maintainer="Tom Manville<tom@kasten.io>"

RUN microdnf -y update openssl-libs

ADD repo-server-controller /repo-server-controller
ENTRYPOINT ["/repo-server-controller"]
