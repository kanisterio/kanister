FROM golang:1.17-alpine3.14 AS builder

ARG kopiaBuildCommit
ARG kopiaRepoOrg

RUN apk add git

WORKDIR /

RUN git clone https://github.com/${kopiaRepoOrg}/kopia.git
WORKDIR /kopia

RUN git checkout ${kopiaBuildCommit}

# Build static executable
RUN GO111MODULE=on CGO_ENABLED=0 GO_EXTLINK_ENABLED=0 go build \
  -ldflags="-s -w -X github.com/kopia/kopia/repo.BuildVersion=${kopiaBuildCommit}" \
  .

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
