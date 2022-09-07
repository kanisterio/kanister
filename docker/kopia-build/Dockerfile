FROM golang:1.19.1-alpine3.16 AS builder

ARG kopiaBuildCommit
ARG kopiaRepoOrg

RUN apk add git

WORKDIR /

RUN git clone https://github.com/${kopiaRepoOrg}/kopia.git

ENV GITHUB_REPOSITORY=https://github.com/${kopiaRepoOrg}/kopia

WORKDIR /kopia

# Build kopia binary from specific commit
RUN git checkout ${kopiaBuildCommit}
# Refers to goreleaser/goreleaser:v1.11.2
COPY --from=docker.io/goreleaser/goreleaser@sha256:b13418f20019fffb29797aff3d03f5e9ca84cea93634f4169a7ed603a95ab198 \
  /usr/bin/goreleaser /usr/bin/goreleaser
RUN goreleaser build --output=. --rm-dist --single-target

RUN adduser -D kopia && addgroup kopia kopia
USER kopia:kopia

COPY --chown=kopia . /kopia

FROM alpine:3.16

WORKDIR /kopia

# Add CA certs
RUN apk add --no-cache --verbose ca-certificates && \
  rm -rf /var/cache/apk/* && \
  adduser -D kopia && addgroup kopia kopia && \
  chown kopia /kopia

USER kopia:kopia

ENTRYPOINT [ "/kopia/kopia" ]

LABEL imageVersion="${imageVersion}"
LABEL kopiaBuildCommit="${kopiaBuildCommit}"

COPY --from=builder --chown=kopia:kopia /kopia/kopia /kopia/kopia
