FROM registry.access.redhat.com/ubi8/ubi-minimal:8.4-212
ARG kan_tools_version="test-version"
LABEL name="kanister-tools" \
    vendor="Kanister" \
    version="${kan_tools_version}" \
    release="${kan_tools_version}" \
    summary="Microservice for application-specific data management for Kubernetes" \
    maintainer="Tom Manville<tom@kasten.io>" \
    description="Kanister tools for application-specific data management"

COPY --from=restic/restic:0.11.0 /usr/bin/restic /usr/local/bin/restic
# kastenhq/kopia alpine-sha-40d45ba image
COPY --from=ghcr.io/kastenhq/kopia@sha256:ad987e7062686fb1c2485e7c1a94268708682152d97f2160a4b93aa514d65b7e \
  /kopia/kopia /usr/local/bin/kopia
COPY LICENSE /licenses/LICENSE

ADD kando /usr/local/bin/
RUN microdnf update && microdnf install shadow-utils httpd-tools && \
  adduser -U kanister -u 1000 && \
  microdnf remove shadow-utils && \
  microdnf clean all

RUN microdnf install yum \
  && yum -y update-minimal --security --sec-severity=Important --sec-severity=Critical \
  && yum clean all \
  && microdnf remove yum \
  && microdnf clean all

CMD [ "/usr/bin/tail", "-f", "/dev/null" ]
