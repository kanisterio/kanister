FROM postgres:9.6-alpine

LABEL maintainer="vkamra@kasten.io"

ENV DEBIAN_FRONTEND noninteractive

USER root

RUN apk -v --update add --no-cache curl python3 py3-pip groff less jq && \
    pip3 install --upgrade pip && \
    pip3 install --upgrade awscli && \
    rm -f /var/cache/apk/*

COPY --from=restic/restic:0.9.5 /usr/bin/restic /usr/local/bin/restic
ADD kando /usr/local/bin/

CMD ["tail", "-f", "/dev/null"]
