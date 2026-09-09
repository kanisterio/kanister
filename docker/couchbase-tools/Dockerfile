FROM couchbase:enterprise-8.0.3@sha256:5b367fe58537a4e3a0445ab8ab95eeee9349ffeafb587c45b07285b9bdc24058

LABEL maintainer="Kanister maintainers<kanister.maintainers@veeam.com>"

# Install Kanister tools
RUN apt-get update && apt-get install libdigest-sha-perl -y
RUN curl https://raw.githubusercontent.com/kanisterio/kanister/master/scripts/get.sh | bash
