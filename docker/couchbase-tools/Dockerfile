FROM couchbase:enterprise-8.0.3@sha256:da4024e7845b928294a72b37ca4f150a9141a00d93b826cff52f4cff0e8f1b0e

LABEL maintainer="Kanister maintainers<kanister.maintainers@veeam.com>"

# Install Kanister tools
RUN apt-get update && apt-get install libdigest-sha-perl -y
RUN curl https://raw.githubusercontent.com/kanisterio/kanister/master/scripts/get.sh | bash
