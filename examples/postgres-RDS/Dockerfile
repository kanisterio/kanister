FROM postgres:15-bookworm@sha256:abb810c09968e4d32fb5f21bced608095215683d0690bc77cb4e4c2a1d8a416f

ENV DEBIAN_FRONTEND noninteractive

USER root

RUN apt-get update && apt-get -y install curl python3-pip python3-venv && \
    python3 -m venv /opt/venv && \
    /opt/venv/bin/pip install --upgrade pip && \
    /opt/venv/bin/pip install --upgrade awscli && \
    apt-get clean

# Add virtual environment to PATH
ENV PATH="/opt/venv/bin:$PATH"

RUN curl https://raw.githubusercontent.com/kanisterio/kanister/master/scripts/get.sh | bash

CMD ["tail", "-f", "/dev/null"]
