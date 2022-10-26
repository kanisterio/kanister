# Build Kopia binary
FROM golang:1.19-bullseye AS builder

ARG kopiaBuildCommit
ARG kopiaRepoOrg
ENV CGO_ENABLED=1 GOEXPERIMENT=boringcrypto GO_EXTLINK_ENABLED=0
RUN apt-get install git

WORKDIR /

RUN git clone https://github.com/${kopiaRepoOrg}/kopia.git

ENV GITHUB_REPOSITORY=https://github.com/${kopiaRepoOrg}/kopia

WORKDIR /kopia

# Build kopia binary from specific commit
RUN git checkout ${kopiaBuildCommit}
RUN GO111MODULE=on GOOS=linux GOARCH=amd64 go build -o kopia \
  -ldflags="-X github.com/kopia/kopia/repo.BuildVersion=$(git show --no-patch --format='%cs-%h') \
            -X github.com/kopia/kopia/repo.BuildInfo=$(git show --no-patch --format='%cI-%H')-${kopiaBuildCommit} \
            -X github.com/kopia/kopia/repo.BuildGitHubRepo=${kopiaRepoOrg}" .

RUN adduser kopia && addgroup kopia kopia
USER kopia:kopia

COPY --chown=kopia . /kopia

FROM debian:bullseye

WORKDIR /kopia

# Add CA certs
RUN apt-get update && apt-get -y install ca-certificates && \
  rm -rf /var/cache/apk/* && \
  adduser kopia && addgroup kopia kopia && \
  chown kopia /kopia

USER kopia:kopia

ENTRYPOINT [ "/kopia/kopia" ]

LABEL imageVersion="${imageVersion}"
LABEL kopiaBuildCommit="${kopiaBuildCommit}"

# Build tools image
FROM registry.access.redhat.com/ubi8/ubi-minimal:8.6-985
ARG kan_tools_version="test-version"
LABEL name="kanister-tools" \
    vendor="Kanister" \
    version="${kan_tools_version}" \
    release="${kan_tools_version}" \
    summary="Microservice for application-specific data management for Kubernetes" \
    maintainer="Tom Manville<tom@kasten.io>" \
    description="Kanister tools for application-specific data management"

COPY --from=restic/restic:0.11.0 /usr/bin/restic /usr/local/bin/restic
COPY --from=builder /kopia/kopia /usr/local/bin/kopia
COPY LICENSE /licenses/LICENSE

ADD kando /usr/local/bin/
RUN microdnf update && microdnf install shadow-utils gzip && \
  adduser -U kanister -u 1000 && \
  microdnf remove shadow-utils && \
  microdnf clean all

CMD [ "/usr/bin/tail", "-f", "/dev/null" ]
