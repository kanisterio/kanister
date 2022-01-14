FROM mcr.microsoft.com/mssql-tools
ENV kubectl_version="v1.18.0"

ADD kando /usr/local/bin/

RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/${kubectl_version}/bin/linux/amd64/kubectl \
    && chmod +x ./kubectl \
    && mv ./kubectl /usr/local/bin/kubectl

CMD [ "/usr/bin/tail", "-f", "/dev/null" ]
