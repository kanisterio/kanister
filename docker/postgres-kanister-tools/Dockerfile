FROM postgres:9.6-alpine
LABEL maintainer="vkamra@kasten.io"

ENV DEBIAN_FRONTEND noninteractive

USER root

RUN apk -v --update add --no-cache python py-pip groff less && \
    pip install --upgrade pip && \
    pip install --upgrade awscli && \
    apk -v --purge del py-pip && \
    rm -f /var/cache/apk/*

COPY --from=kanisterio/kanister-tools:0.15.0 /usr/local/bin/kando /usr/local/bin/kando

CMD ["tail", "-f", "/dev/null"]
