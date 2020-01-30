ARG TOOLS_IMAGE
FROM $TOOLS_IMAGE
RUN apk -v --update add --no-cache mysql-client && \
    rm -f /var/cache/apk/*
