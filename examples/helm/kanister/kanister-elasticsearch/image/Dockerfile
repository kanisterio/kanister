ARG TOOLS_IMAGE
FROM $TOOLS_IMAGE
MAINTAINER Supriya Kharade <supriya@kasten.io>

ADD --chmod=kanister:kanister examples/helm/kanister/kanister-elasticsearch/image/esdump-setup.sh /esdump-setup.sh
RUN chmod +x /esdump-setup.sh && sync && /esdump-setup.sh
