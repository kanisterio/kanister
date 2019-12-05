FROM cassandra:3.11.3

MAINTAINER "Tom Manville <tom@kasten.io>"

# Install restic to take backups
COPY --from=restic/restic:0.9.5 /usr/bin/restic /usr/local/bin/restic

# Install kando 
ADD kando /usr/local/bin/
