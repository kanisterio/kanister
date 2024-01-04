FROM bitnami/cassandra:4.1.3-debian-11-r78

MAINTAINER "Tom Manville <tom@kasten.io>"

# Install restic to take backups
COPY --from=restic/restic:0.16.2 /usr/bin/restic /usr/local/bin/restic

# Install kando 
ADD kando /usr/local/bin/
