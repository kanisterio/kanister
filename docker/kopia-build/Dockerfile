FROM golang:1.17-alpine3.14 AS builder

ARG kopiaBuildCommit
ARG kopiaRepoOrg

RUN apk add git

WORKDIR /

RUN git clone https://github.com/${kopiaRepoOrg}/kopia.git

ENV GITHUB_REPOSITORY=https://github.com/${kopiaRepoOrg}/kopia

WORKDIR /kopia

# Build kopia binary from specific commit
RUN git checkout ${kopiaBuildCommit} && \
  wget https://github.com/goreleaser/goreleaser/releases/download/v1.11.2/goreleaser_1.11.2_x86_64.apk && \
  apk add ./goreleaser_1.11.2_x86_64.apk --allow-untrusted && \
  rm -f goreleaser_1.11.2_x86_64.apk && \
  goreleaser build --output=. --rm-dist --single-target

RUN adduser -D kopia && addgroup kopia kopia
USER kopia:kopia

COPY --chown=kopia . /kopia

FROM alpine:3.14

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
