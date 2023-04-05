FROM postgres:15-bullseye

ENV DEBIAN_FRONTEND noninteractive

USER root

RUN apt-get update && apt-get -y install curl python3-pip && \
    pip3 install --upgrade pip && \
    pip3 install --upgrade awscli && \
    apt-get clean

RUN curl https://raw.githubusercontent.com/kanisterio/kanister/master/scripts/get.sh | bash

CMD ["tail", "-f", "/dev/null"]
