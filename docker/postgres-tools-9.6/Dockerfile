FROM postgres:9.6-alpine

LABEL maintainer="vkamra@kasten.io"

ENV DEBIAN_FRONTEND noninteractive

USER root

RUN apk -v --update add --no-cache curl python py-pip groff less jq && \
    pip install --upgrade pip && \
    pip install --upgrade awscli && \
    apk -v --purge del py-pip && \
    rm -f /var/cache/apk/*

COPY --from=restic/restic:0.9.5 /usr/bin/restic /usr/local/bin/restic
ADD kando /usr/local/bin/

CMD ["tail", "-f", "/dev/null"]
