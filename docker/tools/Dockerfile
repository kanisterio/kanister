FROM registry.access.redhat.com/ubi8/ubi-minimal:8.1

RUN microdnf install git shadow-utils && \
    microdnf clean all && \
    adduser -U kanister

COPY --chown=kanister:kanister --from=restic/restic:0.9.5 /usr/bin/restic /usr/local/bin/restic
COPY --chown=kanister:kanister --from=kanisterio/kopia:alpine-2fb6bc8 /kopia/kopia /usr/local/bin/kopia
ADD --chown=kanister:kanister kando /usr/local/bin/

USER kanister

CMD [ "/usr/bin/tail", "-f", "/dev/null" ]
