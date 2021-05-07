FROM registry.access.redhat.com/ubi8/ubi-minimal:8.1
ARG kan_tools_version="test-version"
LABEL name="kanister-tools" \
    vendor="Kanister" \
    version="${kan_tools_version}" \
    release="${kan_tools_version}" \
    summary="Microservice for application-specific data management for Kubernetes" \
    maintainer="Tom Manville<tom@kasten.io>" \
    description="Kanister tools for application-specific data management"

ENV GODEBUG=asyncpreemptoff=1

COPY --from=restic/restic:0.11.0 /usr/bin/restic /usr/local/bin/restic
# kopia alpine-6c5d58b image
COPY --from=kanisterio/kopia@sha256:f56d34148c4da64ff1490fcc45e4f9230cef989e9b2a5acbfcc51d3410a4066d \
  /kopia/kopia /usr/local/bin/kopia
COPY LICENSE /licenses/LICENSE

ADD kando /usr/local/bin/
RUN microdnf update && microdnf install shadow-utils httpd-tools && \
  adduser -U kanister -u 1000 && \
  microdnf remove shadow-utils && \
  microdnf clean all

CMD [ "/usr/bin/tail", "-f", "/dev/null" ]
