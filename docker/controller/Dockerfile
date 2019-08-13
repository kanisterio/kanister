FROM alpine:3.10
MAINTAINER Tom Manville<tom@kasten.io>
RUN apk -v --update add --no-cache ca-certificates && \
	rm -f /var/cache/apk/*
ADD controller /controller
ENTRYPOINT ["/controller"]
