FROM mcr.microsoft.com/mssql-tools
ENV kubectl_version="v1.18.0"

ADD kando /usr/local/bin/

RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/${kubectl_version}/bin/linux/amd64/kubectl \
    && chmod +x ./kubectl \
    && mv ./kubectl /usr/local/bin/kubectl

COPY --from=ghcr.io/kastenhq/kopia@sha256:ad987e7062686fb1c2485e7c1a94268708682152d97f2160a4b93aa514d65b7e \
  /kopia/kopia /usr/local/bin/kopia

CMD [ "/usr/bin/tail", "-f", "/dev/null" ]
