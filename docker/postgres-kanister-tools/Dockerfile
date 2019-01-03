FROM postgres:9.6-alpine
LABEL maintainer="vkamra@kasten.io"

ENV DEBIAN_FRONTEND noninteractive

USER root

RUN apk -v --update add --no-cache python py-pip groff less && \
    pip install --upgrade pip && \
    pip install --upgrade awscli && \
    apk -v --purge del py-pip && \
    rm -f /var/cache/apk/*

RUN curl https://raw.githubusercontent.com/kanisterio/kanister/master/scripts/get.sh | bash

CMD ["tail", "-f", "/dev/null"]
