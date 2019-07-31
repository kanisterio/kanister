FROM busybox:glibc as restbox

# Get restic executable
ENV RESTIC_VERSION=0.9.5
ENV RESTIC_SHA=08cd75e56a67161e9b16885816f04b2bf1fb5b03bc0677b0ccf3812781c1a2ec
ADD https://github.com/restic/restic/releases/download/v${RESTIC_VERSION}/restic_${RESTIC_VERSION}_linux_amd64.bz2 /
RUN echo "${RESTIC_SHA}  restic_${RESTIC_VERSION}_linux_amd64.bz2" | sha256sum -c \
    && bzip2 -d restic_${RESTIC_VERSION}_linux_amd64.bz2 && mv restic_${RESTIC_VERSION}_linux_amd64 /bin/restic && chmod +x /bin/restic

FROM alpine:3.7
MAINTAINER Tom Manville <tom@kasten.io>

COPY --from=restbox /bin/restic /bin/restic

RUN apk -v --update add --no-cache bash curl groff less mailcap ca-certificates && \
    rm -f /var/cache/apk/*

ADD kando /usr/local/bin/

CMD [ "/usr/bin/tail", "-f", "/dev/null" ]
