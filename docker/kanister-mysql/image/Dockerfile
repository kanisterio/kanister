ARG TOOLS_IMAGE
FROM registry.access.redhat.com/ubi9/ubi:9.1.0-1646.1669627755 as builder

RUN dnf clean all && rm -rf /var/cache/dnf
RUN dnf -y upgrade
# Download the RPM file to avoid timeouts during install
RUN curl -LO https://dev.mysql.com/get/mysql80-community-release-el9-1.noarch.rpm
# Install from the local file
RUN dnf install -y mysql80-community-release-el9-1.noarch.rpm

RUN dnf install -y mysql-community-client

FROM $TOOLS_IMAGE

RUN microdnf -y update && microdnf -y install tar gzip && \
    microdnf clean all

COPY --from=builder /usr/lib64/mysql /usr/lib64/
COPY --from=builder /usr/bin/mysql* /usr/bin/
