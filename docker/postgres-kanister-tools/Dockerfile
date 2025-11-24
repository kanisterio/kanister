# We get tools from tools image
# Tools are not up to date in debian repos
ARG TOOLS_IMAGE
FROM ${TOOLS_IMAGE} AS tools_image

# Actual image base
FROM postgres:18-bookworm@sha256:c09662eac85a25066659be643377e337f072f02b244cf7da1caa8d3440c28c0c

ENV DEBIAN_FRONTEND noninteractive

USER root

COPY docker/postgres-kanister-tools/requirements.txt requirements.txt

RUN apt-get update && apt-get -y install curl python3 groff less jq python3-pip python3-venv && \
    python3 -m venv /opt/venv && \
    /opt/venv/bin/pip install -r requirements.txt && \
    apt-get remove -y python3-setuptools python3-wheel && \
    apt-get clean

# Add virtual environment to PATH
ENV PATH="/opt/venv/bin:$PATH"

# Install restic to take backups
COPY --from=tools_image /usr/local/bin/restic /usr/local/bin/restic
# Update gosu from recent version
COPY --from=tools_image /usr/local/bin/gosu /usr/local/bin/gosu

# Install kando
COPY --from=tools_image /usr/local/bin/kando /usr/local/bin/kando

CMD ["tail", "-f", "/dev/null"]
