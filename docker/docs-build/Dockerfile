FROM ghcr.io/kanisterio/docker-sphinx:0.0.1

ARG GH_VERSION=1.9.2
# add gh (github CLI)
RUN wget https://github.com/cli/cli/releases/download/v${GH_VERSION}/gh_${GH_VERSION}_linux_amd64.deb && \
    dpkg -i gh_1.9.2_linux_amd64.deb

# add aws CLI
RUN curl "https://s3.amazonaws.com/aws-cli/awscli-bundle-1.16.312.zip" -o "awscli-bundle.zip" && \
    unzip awscli-bundle.zip && \
    ./awscli-bundle/install -i /usr/local/aws -b /usr/local/bin/aws
