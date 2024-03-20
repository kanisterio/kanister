# We get tools from tools image
# Tools are not up to date in debian repos
ARG TOOLS_IMAGE
FROM ${TOOLS_IMAGE} AS TOOLS_IMAGE

# Actual image base
FROM postgres:16-bullseye

ENV DEBIAN_FRONTEND noninteractive

USER root

RUN apt-get update && apt-get -y install curl python3 groff less jq python3-pip && \
    pip3 install --upgrade pip setuptools wheel && \
    pip3 install --upgrade awscli && \
    apt-get clean

# Install restic to take backups
COPY --from=TOOLS_IMAGE /usr/local/bin/restic /usr/local/bin/restic
# Update gosu from recent version
COPY --from=TOOLS_IMAGE /usr/local/bin/gosu /usr/local/bin/gosu

ADD kando /usr/local/bin/

CMD ["tail", "-f", "/dev/null"]
