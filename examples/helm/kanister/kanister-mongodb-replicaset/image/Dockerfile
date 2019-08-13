FROM mongo:3.6
MAINTAINER "Tom Manville <tom@kasten.io>"

USER root

ADD ./examples/helm/kanister/kanister-mongodb-replicaset/image /kanister

RUN /kanister/install.sh && rm -rf /kanister && rm -rf /tmp && mkdir /tmp

COPY --from=restic/restic:0.9.5 /usr/bin/restic /usr/local/bin/restic
ADD kando /usr/local/bin/

CMD ["tail", "-f", "/dev/null"]
