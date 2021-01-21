FROM bitnami/cassandra:3.11.8-debian-10-r20

MAINTAINER "Tom Manville <tom@kasten.io>"

# Install restic to take backups
COPY --from=restic/restic:0.11.0 /usr/bin/restic /usr/local/bin/restic

# Install kando 
ADD kando /usr/local/bin/
