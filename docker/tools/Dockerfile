FROM alpine:3.7
MAINTAINER Tom Manville <tom@kasten.io>

RUN apk -v --update add --no-cache bash curl python py-pip groff less mailcap && \
    pip install --upgrade pip && \
    pip install --upgrade awscli && \
    apk -v --purge del py-pip && \
    rm /var/cache/apk/*

RUN curl https://raw.githubusercontent.com/kanisterio/kanister/master/scripts/get_kando.sh | bash

CMD [ "/usr/bin/tail", "-f", "/dev/null" ]
