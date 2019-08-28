FROM bitnami/postgresql:11.5.0-debian-9-r1

USER root

# Explicitly set user/group IDs
RUN useradd -r --gid=0 --uid=1001 postgres

# Install required components for backup
RUN set -x \
	&& apt-get update \
	&& apt-get install -y curl groff lzop pv postgresql-client python3-pip daemontools \
	&& pip3 install --upgrade pip \
	&& hash -r pip3 \
	&& pip3 install wal-e[aws] \
	&& pip3 install awscli \
	&& (cd /usr/local/bin; curl -L https://github.com/wal-g/wal-g/releases/download/v0.1.7/wal-g.linux-amd64.tar.gz | tar -zxv)

# Override base image's CMD script
COPY docker/postgresql/run.sh /

RUN chown postgres:root /run.sh

USER postgres
