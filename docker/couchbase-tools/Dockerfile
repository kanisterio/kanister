FROM couchbase:enterprise-8.0.3@sha256:a4934ba830d2681a227b25d936bfc39a445c1f9e72f4a4bc66f18b59065a4a6a

LABEL maintainer="Kanister maintainers<kanister.maintainers@veeam.com>"

# Install Kanister tools
RUN apt-get update && apt-get install libdigest-sha-perl -y
RUN curl https://raw.githubusercontent.com/kanisterio/kanister/master/scripts/get.sh | bash
