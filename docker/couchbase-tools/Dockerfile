FROM couchbase:enterprise-8.0.3@sha256:ad7c3bd0f490fb788c6918107dd830c0306ae29440525157bb0077af799f3160

LABEL maintainer="Kanister maintainers<kanister.maintainers@veeam.com>"

# Install Kanister tools
RUN apt-get update && apt-get install libdigest-sha-perl -y
RUN curl https://raw.githubusercontent.com/kanisterio/kanister/master/scripts/get.sh | bash
