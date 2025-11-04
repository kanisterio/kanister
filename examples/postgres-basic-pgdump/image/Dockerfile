FROM postgres:9.6-alpine
LABEL maintainer="kanister.maintainers@veeam.com"

ENV DEBIAN_FRONTEND noninteractive

USER root

ADD . /install

RUN /install/install.sh && rm -rf /install && rm -rf /tmp && mkdir /tmp
