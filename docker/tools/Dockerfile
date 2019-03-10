FROM busybox:glibc as restbox

# Get restic executable
ENV RESTIC_VERSION=0.9.4
ADD https://github.com/restic/restic/releases/download/v${RESTIC_VERSION}/restic_${RESTIC_VERSION}_linux_amd64.bz2 /
RUN bzip2 -d restic_${RESTIC_VERSION}_linux_amd64.bz2 && mv restic_${RESTIC_VERSION}_linux_amd64 /bin/restic && chmod +x /bin/restic

FROM alpine:3.7
MAINTAINER Tom Manville <tom@kasten.io>

COPY --from=restbox /bin/restic /bin/restic

RUN apk -v --update add --no-cache bash curl groff less mailcap ca-certificates && \
    rm -f /var/cache/apk/*

RUN curl https://raw.githubusercontent.com/kanisterio/kanister/master/scripts/get.sh | bash

CMD [ "/usr/bin/tail", "-f", "/dev/null" ]
