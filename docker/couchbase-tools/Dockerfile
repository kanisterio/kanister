FROM couchbase:enterprise-8.0.2@sha256:c118dfa489399b85e4304f5d3133d2994d2a226fe81170698a02e6c6875636c3

LABEL maintainer="Kanister maintainers<kanister.maintainers@veeam.com>"

# Install Kanister tools
RUN apt-get update && apt-get install libdigest-sha-perl -y
RUN curl https://raw.githubusercontent.com/kanisterio/kanister/master/scripts/get.sh | bash
