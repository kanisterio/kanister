FROM mongo:3.6
MAINTAINER "Tom Manville <tom@kasten.io>"

COPY --from=restic/restic:0.9.5 /usr/bin/restic /usr/local/bin/restic
ADD kando /usr/local/bin/
