FROM postgres:16-bullseye

ENV DEBIAN_FRONTEND noninteractive

USER root

RUN apt-get update && apt-get -y install curl python3 groff less jq python3-pip && \
    pip3 install --upgrade pip && \
    pip3 install --upgrade awscli && \
    apt-get clean

COPY --from=restic/restic:0.16.2 /usr/bin/restic /usr/local/bin/restic
ADD kando /usr/local/bin/

CMD ["tail", "-f", "/dev/null"]
