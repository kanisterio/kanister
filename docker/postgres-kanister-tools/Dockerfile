FROM postgres:12.0-alpine
LABEL maintainer="vkamra@kasten.io"

ENV DEBIAN_FRONTEND noninteractive

USER root

RUN apk -v --update add --no-cache curl python3 groff less jq py-pip && \
    pip3 install --upgrade pip && \
    pip3 install --upgrade awscli && \
    rm -f /var/cache/apk/*

COPY --from=restic/restic:0.11.0 /usr/bin/restic /usr/local/bin/restic
ADD kando /usr/local/bin/

CMD ["tail", "-f", "/dev/null"]
