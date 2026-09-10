FROM couchbase:enterprise-8.0.3@sha256:c4254e37f5a5db3d13eb990d89862765db44b4f96b6283190c2e6c8a20d760d7

LABEL maintainer="Kanister maintainers<kanister.maintainers@veeam.com>"

# Install Kanister tools
RUN apt-get update && apt-get install libdigest-sha-perl -y
RUN curl https://raw.githubusercontent.com/kanisterio/kanister/master/scripts/get.sh | bash
