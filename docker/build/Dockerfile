FROM golang:1.12.4-alpine3.9
MAINTAINER Tom Manville <tom@kasten.io>

RUN apk add --update --no-cache ca-certificates bash git \
 && update-ca-certificates \
 && rm -rf /var/cache/apk/*

COPY --from=bitnami/kubectl:1.13.4 /opt/bitnami/kubectl/bin/kubectl /usr/local/bin/

COPY --from=goreleaser/goreleaser:v0.112.2 /bin/goreleaser /usr/local/bin/

ENV CGO_ENABLED=0 \
    GO111MODULE="on" \
    GOROOT="/usr/local/go" \
    GO_EXTLINK_ENABLED=0 \
    PATH="/usr/local/go/bin:${PATH}" 
