FROM alpine:3.10.5

LABEL maintainer="Tom Manville<tom@kasten.io>"

RUN apk add --update --no-cache ca-certificates bash git docker jq \
    && update-ca-certificates \
    && rm -rf /var/cache/apk/*

COPY --from=golang:1.17.6-alpine3.15 /usr/local/go/ /usr/local/go/

COPY --from=bitnami/kubectl:1.17 /opt/bitnami/kubectl/bin/kubectl /usr/local/bin/

COPY --from=goreleaser/goreleaser:v1.2.4 /usr/local/bin/goreleaser /usr/local/bin/

COPY --from=alpine/helm:3.1.0 /usr/bin/helm /usr/local/bin/

COPY --from=golangci/golangci-lint:v1.23.7 /usr/bin/golangci-lint /usr/local/bin/

RUN wget -O /usr/local/bin/kind https://github.com/kubernetes-sigs/kind/releases/download/v0.11.1/kind-linux-amd64 \
    && chmod +x /usr/local/bin/kind

ENV CGO_ENABLED=0 \
    GO111MODULE="on" \
    GOROOT="/usr/local/go" \
    GOCACHE=/go/.cache/go-build \
    GO_EXTLINK_ENABLED=0 \
    PATH="/usr/local/go/bin:${PATH}"
