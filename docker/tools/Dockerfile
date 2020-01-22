FROM alpine:3.10

RUN apk -v --update add --no-cache bash curl groff less mailcap ca-certificates && \
    rm -f /var/cache/apk/*

COPY --from=restic/restic:0.9.5 /usr/bin/restic /usr/local/bin/restic
COPY --from=kanisterio/kopia:alpine-9849fea /kopia/kopia /usr/local/bin/kopia
ADD kando /usr/local/bin/

CMD [ "/usr/bin/tail", "-f", "/dev/null" ]
