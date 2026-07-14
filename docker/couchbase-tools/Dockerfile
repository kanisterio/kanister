FROM couchbase:enterprise-8.0.2@sha256:e3773914cdfe8ec8014f0c6eb81f6eda70030de3ba86ec81402b2a82d0df39fc

LABEL maintainer="Kanister maintainers<kanister.maintainers@veeam.com>"

# Install Kanister tools
RUN apt-get update && apt-get install libdigest-sha-perl -y
RUN curl https://raw.githubusercontent.com/kanisterio/kanister/master/scripts/get.sh | bash
