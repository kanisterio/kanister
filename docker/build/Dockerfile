FROM golang:1.19-bullseye

LABEL maintainer="Tom Manville<tom@kasten.io>"

RUN apt-get update && apt-get -y install apt-transport-https ca-certificates bash git gnupg2 software-properties-common curl jq wget \
    && update-ca-certificates

RUN curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg \
    && echo "deb [arch=amd64 signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/debian $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list

RUN apt update && apt install -y docker-ce docker-ce-cli containerd.io \
    && apt-get clean

COPY --from=bitnami/kubectl:1.26 /opt/bitnami/kubectl/bin/kubectl /usr/local/bin/

COPY --from=goreleaser/goreleaser:v1.12.3 /usr/bin/goreleaser /usr/local/bin/

COPY --from=alpine/helm:3.2.0 /usr/bin/helm /usr/local/bin/

COPY --from=golangci/golangci-lint:v1.50 /usr/bin/golangci-lint /usr/local/bin/

RUN wget -O /usr/local/bin/kind https://github.com/kubernetes-sigs/kind/releases/download/v0.18.0/kind-linux-amd64 \
    && chmod +x /usr/local/bin/kind

# Adding CRD documentation generation tool.
RUN GOBIN=/usr/local/bin go install github.com/ahmetb/gen-crd-api-reference-docs@v0.3.0

ENV CGO_ENABLED=0 \
    GO111MODULE="on" \
    GOROOT="/usr/local/go" \
    GOCACHE=/go/.cache/go-build \
    GO_EXTLINK_ENABLED=0 \
    PATH="/usr/local/go/bin:${PATH}"
