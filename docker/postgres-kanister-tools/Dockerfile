# We get tools from tools image
# Tools are not up to date in debian repos
ARG TOOLS_IMAGE
FROM ${TOOLS_IMAGE} AS TOOLS_IMAGE

# Actual image base
FROM postgres:16-bullseye

ENV DEBIAN_FRONTEND noninteractive

USER root

COPY docker/postgres-kanister-tools/requirements.txt requirements.txt

RUN apt-get update && apt-get -y install curl python3 groff less jq python3-pip && \
    pip3 install -r requirements.txt && \
    apt-get remove -y python3-setuptools python3-wheel && \
    apt-get clean

# Install restic to take backups
COPY --from=TOOLS_IMAGE /usr/local/bin/restic /usr/local/bin/restic
# Update gosu from recent version
COPY --from=TOOLS_IMAGE /usr/local/bin/gosu /usr/local/bin/gosu

# Install kando
COPY --from=TOOLS_IMAGE /usr/local/bin/kando /usr/local/bin/kando

CMD ["tail", "-f", "/dev/null"]
