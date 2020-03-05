FROM ubuntu:18.04

USER root

# Installs base packages via apt-get
COPY 00-base/ /build_image/00-base/
RUN /build_image/00-base/install.sh && rm -rf /tmp && mkdir /tmp

# Install other tools
COPY 01-tools/ /build_image/01-tools/
RUN /build_image/01-tools/install.sh

# Install Go and other tools via `go get`
COPY 02-go/ /build_image/02-go/
RUN /build_image/02-go/install.sh

#COPY --from=bitnami/kubectl:1.13.4 /opt/bitnami/kubectl/bin/kubectl /usr/local/bin/

COPY --from=goreleaser/goreleaser:v0.116.0 /bin/goreleaser /usr/local/bin/

# COPY --from=alpine/helm:3.0.0 /usr/bin/helm /usr/local/bin/

# RUN wget -O /usr/local/bin/kind https://github.com/kubernetes-sigs/kind/releases/download/v0.5.1/kind-linux-amd64 \
#     && chmod +x /usr/local/bin/kind

# Include a copy of the Dockerfile in the image.
COPY Dockerfile /build_image

ENV CGO_ENABLED=0 \
    GO111MODULE="on" \
    GOROOT="/usr/local/go" \
    GOCACHE=/go/.cache/go-build \
    GO_EXTLINK_ENABLED=0 \
    PATH="/usr/local/go/bin:${PATH}" 

CMD [ "/bin/bash", "-l" ]
