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

COPY --from=goreleaser/goreleaser:v0.116.0 /bin/goreleaser /usr/local/bin/

# Include a copy of the Dockerfile in the image.
COPY Dockerfile /build_image

ENV CGO_ENABLED=0 \
    GO111MODULE="on" \
    GOROOT="/usr/local/go" \
    GOCACHE=/go/.cache/go-build \
    GO_EXTLINK_ENABLED=0 \
    PATH="/usr/local/go/bin:${PATH}" 

CMD [ "/bin/bash", "-l" ]
