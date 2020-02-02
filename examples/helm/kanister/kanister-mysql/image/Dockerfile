ARG TOOLS_IMAGE
FROM registry.access.redhat.com/ubi8/ubi:8.1 as builder
RUN dnf install -y https://dev.mysql.com/get/mysql80-community-release-el8-1.noarch.rpm
RUN dnf install -y mysql-community-client

FROM $TOOLS_IMAGE

RUN microdnf update && microdnf install tar gzip && \
    microdnf clean all

COPY --from=builder /usr/lib64/mysql /usr/lib64/
COPY --from=builder /usr/bin/mysql* /usr/bin/
