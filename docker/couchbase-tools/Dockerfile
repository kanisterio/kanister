FROM couchbase:enterprise-7.6.2

MAINTAINER "Tom Manville <tom@kasten.io>"

# Install Kanister tools
RUN apt-get update && apt-get install libdigest-sha-perl -y
RUN curl https://raw.githubusercontent.com/kanisterio/kanister/master/scripts/get.sh | bash
