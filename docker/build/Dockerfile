FROM golang:1.12-alpine3.10
MAINTAINER Tom Manville <tom@kasten.io>

RUN apk add --update --no-cache ca-certificates bash git docker jq \
    && update-ca-certificates \
    && rm -rf /var/cache/apk/*

COPY --from=bitnami/kubectl:1.13.4 /opt/bitnami/kubectl/bin/kubectl /usr/local/bin/

COPY --from=goreleaser/goreleaser:v0.116.0 /bin/goreleaser /usr/local/bin/

RUN wget -O /usr/local/bin/kind https://github.com/kubernetes-sigs/kind/releases/download/v0.5.1/kind-linux-amd64 \
    && chmod +x /usr/local/bin/kind

ENV CGO_ENABLED=0 \
    GO111MODULE="on" \
    GOROOT="/usr/local/go" \
    GOCACHE=/go/.cache/go-build \
    GO_EXTLINK_ENABLED=0 \
    PATH="/usr/local/go/bin:${PATH}" 
