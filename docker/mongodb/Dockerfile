FROM bitnami/mongodb:4.0.12-debian-9-r22

MAINTAINER "Tom Manville <tom@kasten.io>"

# Install restic to take backups
COPY --from=restic/restic:0.9.5 /usr/bin/restic /usr/local/bin/restic

# Install kando
ADD kando /usr/local/bin/
