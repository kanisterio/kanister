ARG TOOLS_IMAGE
FROM ${TOOLS_IMAGE} AS TOOLS_IMAGE

FROM alpine:3.11.3
COPY --from=TOOLS_IMAGE /usr/local/bin/restic /usr/local/bin/restic
COPY --from=TOOLS_IMAGE /usr/local/bin/kopia /usr/local/bin/kopia
ADD kando /usr/local/bin/

ADD docker/kanister-elasticsearch/image/esdump-setup.sh /esdump-setup.sh
RUN chmod +x /esdump-setup.sh && sync && /esdump-setup.sh

CMD [ "/usr/bin/tail", "-f", "/dev/null" ]
