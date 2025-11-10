FROM postgres:15-bookworm

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
