FROM postgres:9.6-alpine
LABEL maintainer="vkamra@kasten.io"

ENV DEBIAN_FRONTEND noninteractive

USER root

COPY --from=kanisterio/kanister-tools:0.11.0 /usr/local/bin/kando /usr/local/bin/kando

CMD ["tail", "-f", "/dev/null"]
