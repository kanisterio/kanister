# We get tools from tools image
# Tools are not up to date in debian repos
ARG TOOLS_IMAGE
FROM ${TOOLS_IMAGE} AS TOOLS_IMAGE

# Actual image base
FROM bitnamilegacy/cassandra:5.0.4@sha256:215334c52eb4f81ec90bedf108a762e6a806a4cb0dfe615726d6e6f1b4fb2fbb

LABEL maintainer="Kanister maintainers<kanister.maintainers@veeam.com>"

# Install restic to take backups
COPY --from=TOOLS_IMAGE /usr/local/bin/restic /usr/local/bin/restic
# Update gosu from recent version
COPY --from=TOOLS_IMAGE /usr/local/bin/gosu /usr/local/bin/gosu

# Install kando
COPY --from=TOOLS_IMAGE /usr/local/bin/kando /usr/local/bin/kando
