FROM registry.access.redhat.com/ubi8/ubi-minimal:8.1
ARG kan_tools_version="test-version"
LABEL name="kanister-tools" \
    vendor="Kanister" \
    version="${kan_tools_version}" \
    release="${kan_tools_version}" \
    summary="Microservice for application-specific data management for Kubernetes" \
    maintainer="Tom Manville<tom@kasten.io>"
    description="Kanister tools for application-specific data management"

COPY --from=restic/restic:0.9.5 /usr/bin/restic /usr/local/bin/restic
COPY --from=kanisterio/kopia:alpine-b46ca3a /kopia/kopia /usr/local/bin/kopia
ADD kando /usr/local/bin/

CMD [ "/usr/bin/tail", "-f", "/dev/null" ]
