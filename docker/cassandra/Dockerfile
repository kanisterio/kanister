# We get tools from tools image
# Tools are not up to date in debian repos
ARG TOOLS_IMAGE
FROM ${TOOLS_IMAGE} AS TOOLS_IMAGE

# Actual image base
FROM bitnami/cassandra:4.1.6

MAINTAINER "Tom Manville <tom@kasten.io>"

# Install restic to take backups
COPY --from=TOOLS_IMAGE /usr/local/bin/restic /usr/local/bin/restic
# Update gosu from recent version
COPY --from=TOOLS_IMAGE /usr/local/bin/gosu /usr/local/bin/gosu

# Install kando
COPY --from=TOOLS_IMAGE /usr/local/bin/kando /usr/local/bin/kando
