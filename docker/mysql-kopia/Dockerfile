# Build Kopia binary
FROM bitnami/golang:1.19.2-debian-11-r0 AS builder

ARG kopia_build_commit=master
ARG kopia_repo_org=kopia
ARG kanister_repo_org=kanisterio
ARG kanister_build_commit=master
ENV CGO_ENABLED=1 GOEXPERIMENT=boringcrypto GO_EXTLINK_ENABLED=0
RUN apt-get update && apt-get install -y git glibc-source

WORKDIR /

RUN git clone https://github.com/${kopia_repo_org}/kopia.git
RUN git clone https://github.com/${kanister_repo_org}/kanister.git

ENV GITHUB_REPOSITORY=https://github.com/${kanister_repo_org}/kanister

WORKDIR /kanister

RUN git checkout ${kanister_build_commit}
RUN go build -o kando cmd/kando/main.go
RUN chmod a+x ./kando

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
  chown kopia /kopia && \
  chown kopia /kanister

USER kopia:kopia

# Build mysql-kopia image
FROM bitnami/mysql:8.0.32-debian-11-r14

# Install kopia to take backups
COPY --from=builder /kopia/kopia /usr/local/bin/kopia
COPY --from=builder /kanister/kando /usr/local/bin/kando
