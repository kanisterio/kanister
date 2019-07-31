FROM mongo:3.6
MAINTAINER "Tom Manville <tom@kasten.io>"

USER root

ADD ./examples/helm/kanister/kanister-mongodb-replicaset/image /kanister

RUN /kanister/install.sh && rm -rf /kanister && rm -rf /tmp && mkdir /tmp

RUN curl https://raw.githubusercontent.com/kanisterio/kanister/master/scripts/get.sh | bash

CMD ["tail", "-f", "/dev/null"]
