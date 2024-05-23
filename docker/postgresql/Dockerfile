FROM bitnami/postgresql:16

USER root

# Explicitly set user/group IDs
RUN useradd -r --gid=0 --uid=1001 postgres

# Install required components for backup
RUN set -x \
	&& apt-get update \
	&& apt-get install -y curl groff lzop pv postgresql-client python3-pip daemontools \
	&& pip3 install --break-system-packages --upgrade pip \
	&& hash -r pip3 \
	&& pip3 install --break-system-packages wal-e[aws] \
	&& pip3 install --break-system-packages awscli

USER postgres
