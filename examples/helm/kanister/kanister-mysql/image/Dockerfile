ARG TOOLS_IMAGE
FROM registry.access.redhat.com/ubi8/ubi:8.1 as builder
RUN dnf install -y https://dev.mysql.com/get/mysql80-community-release-el8-1.noarch.rpm
RUN dnf install -y mysql-community-client

FROM $TOOLS_IMAGE

COPY --from=builder --chown=kanister:kanister /usr/lib64/mysql /usr/lib64/
COPY --from=builder --chown=kanister:kanister /usr/bin/mysql* /usr/bin/
