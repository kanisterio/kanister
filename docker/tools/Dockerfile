FROM registry.access.redhat.com/ubi8/ubi-minimal:8.1

COPY --from=restic/restic:0.9.5 /usr/bin/restic /usr/local/bin/restic
COPY --from=kanisterio/kopia:alpine-b46ca3a /kopia/kopia /usr/local/bin/kopia
ADD kando /usr/local/bin/
RUN microdnf update && microdnf install shadow-utils && \
  adduser -U kio -u 1000 && \
  microdnf remove shadow-utils && \
  microdnf clean all

CMD [ "/usr/bin/tail", "-f", "/dev/null" ]
