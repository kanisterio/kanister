FROM postgres:14-bullseye
LABEL maintainer="vkamra@kasten.io"

ENV DEBIAN_FRONTEND noninteractive

USER root

RUN apt-get update && apt-get -y install curl python3 groff less jq python3-pip && \
    pip3 install --upgrade pip && \
    pip3 install --upgrade awscli && \
    apt-get clean

COPY --from=restic/restic:0.11.0 /usr/bin/restic /usr/local/bin/restic
ADD kando /usr/local/bin/

CMD ["tail", "-f", "/dev/null"]
