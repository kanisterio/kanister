FROM phusion/baseimage:0.9.19
MAINTAINER "Tom Manville <tom@kasten.io>"

USER root

ADD . /kanister

RUN /kanister/install.sh && rm -rf /kanister && rm -rf /tmp && mkdir /tmp

RUN curl https://raw.githubusercontent.com/kanisterio/kanister/master/scripts/get.sh | bash

CMD ["tail", "-f", "/dev/null"]
