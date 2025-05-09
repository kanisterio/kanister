FROM golang:1.24.3-bullseye@sha256:9cf2ca5977b21567c301d2528fc329b1ef1a6e391a50c48f4383feb9019b3a9c
LABEL maintainer="Kanister maintainers<kanister.maintainers@veeam.com>"

ARG TARGETPLATFORM

RUN apt-get update && apt-get -y install \
    apt-transport-https bash ca-certificates curl git gnupg2 jq protobuf-compiler software-properties-common wget \
    && update-ca-certificates

RUN curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg \
    && echo "deb [signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/debian $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list

RUN apt update && apt install -y docker-ce docker-ce-cli containerd.io \
    && apt-get clean

COPY --from=bitnami/kubectl:1.26 /opt/bitnami/kubectl/bin/kubectl /usr/local/bin/

COPY --from=goreleaser/goreleaser:v1.26.2 /usr/bin/goreleaser /usr/local/bin/

COPY --from=alpine/helm:3.12.2 /usr/bin/helm /usr/local/bin/

COPY --from=golangci/golangci-lint:v1.64.7 /usr/bin/golangci-lint /usr/local/bin/

RUN wget -O /usr/local/bin/kind \
      https://github.com/kubernetes-sigs/kind/releases/download/v0.18.0/kind-$(echo $TARGETPLATFORM | tr / -) \
    && chmod +x /usr/local/bin/kind

RUN git config --global --add safe.directory /go/src/github.com/kanisterio/kanister

# Adding CRD documentation generation tool.
RUN GOBIN=/usr/local/bin go install github.com/ahmetb/gen-crd-api-reference-docs@v0.3.0

RUN apt-get update && apt-get install -y pip

COPY requirements.txt requirements.txt
RUN pip install -r requirements.txt

RUN apt-get install -y vim

ENV CGO_ENABLED=0 \
    GO111MODULE="on" \
    GOROOT="/usr/local/go" \
    GOCACHE=/go/.cache/go-build \
    GO_EXTLINK_ENABLED=0 \
    PATH="/usr/local/go/bin:${PATH}"
