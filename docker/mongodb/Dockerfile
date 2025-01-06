# We get tools from tools image
# Tools are not up to date in debian repos
ARG TOOLS_IMAGE
FROM ${TOOLS_IMAGE} AS TOOLS_IMAGE

FROM bitnami/mongodb:7.0-debian-12

LABEL maintainer="Kanister maintainers <kanister.maintainers@veeam.com>"

# Update gosu from recent version
COPY --from=TOOLS_IMAGE /usr/local/bin/gosu /usr/local/bin/gosu

# Install kando
COPY --from=TOOLS_IMAGE /usr/local/bin/kando /usr/local/bin/kando
