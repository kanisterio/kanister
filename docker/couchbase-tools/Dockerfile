FROM couchbase:enterprise-8.0.3@sha256:db0e1cdcaf4bf4a86f7ba65857c4693ac3b5600900ec59e796fac8e75ee5293d

LABEL maintainer="Kanister maintainers<kanister.maintainers@veeam.com>"

# Install Kanister tools
RUN apt-get update && apt-get install libdigest-sha-perl -y
RUN curl https://raw.githubusercontent.com/kanisterio/kanister/master/scripts/get.sh | bash
