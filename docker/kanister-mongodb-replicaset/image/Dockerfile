FROM mongo:3.6
LABEL maintainer="Tom Manville <tom@kasten.io>"

USER root

ADD . /kanister

RUN /kanister/install.sh && rm -rf /kanister && rm -rf /tmp && mkdir /tmp

COPY --from=restic/restic:0.16.4 /usr/bin/restic /usr/local/bin/restic

CMD ["tail", "-f", "/dev/null"]
