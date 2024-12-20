FROM mongo:3.6
LABEL maintainer="Kanister maintainers <kanister.maintainers@veeam.com>"

USER root

ADD . /kanister

RUN /kanister/install.sh && rm -rf /kanister && rm -rf /tmp && mkdir /tmp

COPY --from=restic/restic:0.16.4 /usr/bin/restic /usr/local/bin/restic

CMD ["tail", "-f", "/dev/null"]
