FROM couchbase:enterprise-8.0.3@sha256:992138aa1b546f41e0d823f1bd64676868106579e14cbc02dbb5d65bdcfcc334

LABEL maintainer="Kanister maintainers<kanister.maintainers@veeam.com>"

# Install Kanister tools
RUN apt-get update && apt-get install libdigest-sha-perl -y
RUN curl https://raw.githubusercontent.com/kanisterio/kanister/master/scripts/get.sh | bash
