# Build Kopia binary
FROM golang:1.19-bullseye AS builder

ARG kopia_build_commit=master
ARG kopia_repo_org=kopia
ENV CGO_ENABLED=1 GOEXPERIMENT=boringcrypto GO_EXTLINK_ENABLED=0
RUN apt-get install git

WORKDIR /

RUN git clone https://github.com/${kopia_repo_org}/kopia.git

ENV GITHUB_REPOSITORY=https://github.com/${kopia_repo_org}/kopia

WORKDIR /kopia

# Build kopia binary from specific commit
RUN git checkout ${kopia_build_commit}
RUN GO111MODULE=on GOOS=linux GOARCH=amd64 go build -o kopia \
  -ldflags="-X github.com/kopia/kopia/repo.BuildVersion=$(git show --no-patch --format='%cs-%h') \
            -X github.com/kopia/kopia/repo.BuildInfo=$(git show --no-patch --format='%cI-%H')-${kopia_build_commit} \
            -X github.com/kopia/kopia/repo.BuildGitHubRepo=${kopia_repo_org}" .

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

# Build mysql-kopia image
FROM bitnami/mysql:8.0.32-debian-11-r14

RUN apt-get update && apt-get install -y glibc-source
# Install kopia to take backups
COPY --from=builder /kopia/kopia /usr/local/bin/kopia

# Install kando
ADD kando /usr/local/bin/
