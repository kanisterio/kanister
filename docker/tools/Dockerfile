# Build Kopia binary
FROM golang:1.24.3-bullseye@sha256:9cf2ca5977b21567c301d2528fc329b1ef1a6e391a50c48f4383feb9019b3a9c AS builder

ARG kopia_build_commit=master
ARG kopia_repo_org=kopia
ARG restic_vsn=v0.16.5
ARG gosu_vsn=1.17
ENV CGO_ENABLED=1 GOEXPERIMENT=boringcrypto GO_EXTLINK_ENABLED=0
RUN apt-get install git

# Build restic binary from source - released version
# This will allow us to bring in security fixes without relying on the official
# image which is released once every quarter
WORKDIR /

RUN git clone https://github.com/restic/restic.git

ENV GITHUB_REPOSITORY=https://github.com/restic/restic

WORKDIR /restic

RUN git checkout ${restic_vsn} && \
    echo 'package main' > cmd/restic/fipsonly.go && \
    echo 'import _ "crypto/tls/fipsonly"' >> cmd/restic/fipsonly.go
# use debug flag to preserve symbols
RUN go run build.go --tags debug

# Build restic binary from source - released version
# This will allow us to bring in security fixes more up to date then apt repos
WORKDIR /

RUN git clone https://github.com/tianon/gosu.git

ENV GITHUB_REPOSITORY=https://github.com/tianon/gosu

WORKDIR /gosu

RUN git checkout ${gosu_vsn}
RUN go build -o gosu

# Build kopia binary from specific commit
WORKDIR /

RUN git clone https://github.com/${kopia_repo_org}/kopia.git

ENV GITHUB_REPOSITORY=https://github.com/${kopia_repo_org}/kopia

WORKDIR /kopia

RUN git checkout ${kopia_build_commit} && \
    echo 'package main' > fipsonly.go && \
    echo 'import _ "crypto/tls/fipsonly"' >> fipsonly.go

RUN GO111MODULE=on GOOS=linux GOARCH=amd64 go build -o kopia \
  -ldflags="-X github.com/kopia/kopia/repo.BuildVersion=$(git show --no-patch --format='%cs-%h') \
            -X github.com/kopia/kopia/repo.BuildInfo=$(git show --no-patch --format='%cI-%H')-${kopia_build_commit} \
            -X github.com/kopia/kopia/repo.BuildGitHubRepo=${kopia_repo_org}" .

RUN adduser kopia && addgroup kopia kopia
USER kopia:kopia

COPY --chown=kopia . /kopia

FROM debian:bullseye@sha256:2a7f95bcf104c8410bf4d3b13c52f6e0e4334bb2edf8d80c7f9881e49447effe

WORKDIR /kopia

# Add CA certs
RUN apt-get update && apt-get -y install ca-certificates && \
  rm -rf /var/cache/apk/* && \
  adduser kopia && addgroup kopia kopia && \
  chown kopia /kopia

USER kopia:kopia

# Build tools image
FROM registry.access.redhat.com/ubi9/ubi-minimal:9.5@sha256:e1c4703364c5cb58f5462575dc90345bcd934ddc45e6c32f9c162f2b5617681c
ARG kan_tools_version="test-version"
LABEL name="kanister-tools" \
    vendor="Kanister" \
    version="${kan_tools_version}" \
    release="${kan_tools_version}" \
    summary="Operator for data protection workflow management on Kubernetes" \
    maintainer="Kanister maintainers<kanister.maintainers@veeam.com>" \
    description="Tools for application-specific data protection"

COPY --from=builder /restic/restic /usr/local/bin/restic
COPY --from=builder /gosu/gosu /usr/local/bin/gosu
COPY --from=builder /kopia/kopia /usr/local/bin/kopia
COPY LICENSE /licenses/LICENSE

ADD kando /usr/local/bin/
RUN microdnf -y update && microdnf -y install shadow-utils gzip && \
  adduser -U kanister -u 1000 && \
  microdnf -y remove shadow-utils && \
  microdnf clean all

RUN setcap cap_chown,cap_fowner,cap_dac_override+iep /usr/local/bin/kopia

CMD [ "/usr/bin/tail", "-f", "/dev/null" ]
