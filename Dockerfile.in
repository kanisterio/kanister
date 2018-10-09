FROM ARG_FROM
MAINTAINER Tom Manville<tom@kasten.io>
ADD ARG_SOURCE_BIN /ARG_BIN
RUN apk -v --update add --no-cache ca-certificates && \
	rm -f /var/cache/apk/*
ENTRYPOINT ["/ARG_BIN"]
