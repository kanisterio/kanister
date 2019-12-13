FROM alpine:3.10
MAINTAINER Tom Manville <tom@kasten.io>

RUN apk -v --update add --no-cache bash curl groff less mailcap ca-certificates && \
    rm -f /var/cache/apk/*

COPY --from=restic/restic:0.9.5 /usr/bin/restic /usr/local/bin/restic
COPY --from=kopia/kopia:alpine-3f721aa /kopia/kopia /usr/local/bin/kopia
ADD kando /usr/local/bin/

CMD [ "/usr/bin/tail", "-f", "/dev/null" ]
