ARG TOOLS_IMAGE
FROM ${TOOLS_IMAGE} AS TOOLS_IMAGE

FROM mcr.microsoft.com/mssql-tools
ENV kubectl_version="v1.18.0"

# Install kando
COPY --from=TOOLS_IMAGE /usr/local/bin/kando /usr/local/bin/kando

RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/${kubectl_version}/bin/linux/amd64/kubectl \
    && chmod +x ./kubectl \
    && mv ./kubectl /usr/local/bin/kubectl

CMD [ "/usr/bin/tail", "-f", "/dev/null" ]
