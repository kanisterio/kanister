# We get tools from tools image
# Tools are not up to date in debian repos
ARG TOOLS_IMAGE
FROM ${TOOLS_IMAGE} AS TOOLS_IMAGE

FROM bitnami/mongodb:7.0-debian-12

LABEL maintainer="Tom Manville <tom@kasten.io>"

# Update gosu from recent version
COPY --from=TOOLS_IMAGE /usr/local/bin/gosu /usr/local/bin/gosu

# Install kando
ADD kando /usr/local/bin/
