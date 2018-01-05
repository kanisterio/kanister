FROM postgres:9.6-alpine
LABEL maintainer="vkamra@kasten.io"

ENV DEBIAN_FRONTEND noninteractive

USER root

ADD . /install

RUN /install/install.sh && rm -rf /install && rm -rf /tmp && mkdir /tmp
