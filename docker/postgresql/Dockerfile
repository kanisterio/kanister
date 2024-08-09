FROM bitnami/postgresql:16

USER root

# Explicitly set user/group IDs
RUN useradd -r --gid=0 --uid=1001 postgres

# Install required components for backup
COPY docker/postgresql/requirements.txt requirements.txt

RUN set -x \
	&& apt-get update \
	&& apt-get install -y curl groff lzop pv postgresql-client python3-pip daemontools \
	&& pip3 install --break-system-packages -r requirements.txt

USER postgres
