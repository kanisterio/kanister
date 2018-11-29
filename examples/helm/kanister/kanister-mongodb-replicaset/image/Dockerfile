FROM mongo:3.6
MAINTAINER "Tom Manville <tom@kasten.io>"

USER root

ADD . /kanister

RUN /kanister/install.sh && rm -rf /kanister && rm -rf /tmp && mkdir /tmp

COPY --from=kanisterio/kanister-tools:0.14.0 /usr/local/bin/kando /usr/local/bin/kando

CMD ["tail", "-f", "/dev/null"]
