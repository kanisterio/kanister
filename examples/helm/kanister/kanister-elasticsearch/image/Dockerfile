ARG TOOLS_IMAGE
FROM $TOOLS_IMAGE

ADD examples/helm/kanister/kanister-elasticsearch/image/esdump-setup.sh /esdump-setup.sh
RUN chmod +x /esdump-setup.sh && sync && /esdump-setup.sh

