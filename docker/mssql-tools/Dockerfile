ARG TOOLS_IMAGE
FROM ${TOOLS_IMAGE} AS tools_image

FROM ubuntu:noble@sha256:cd1dba651b3080c3686ecf4e3c4220f026b521fb76978881737d24f200828b2b

# Declare build arguments for this stage
ARG TARGETARCH
ARG kubectl_version="v1.18.0"

# Validate supported architecture
RUN case "${TARGETARCH}" in \
      amd64) ;; \
      arm64) ;; \
      *) printf 'Unsupported platform: %s\n' "${TARGETARCH}" && exit 1 ;; \
    esac

# Install dependencies
RUN export DEBIAN_FRONTEND=noninteractive && \
    apt-get update && \
    apt-get install -y curl ca-certificates gnupg lsb-release && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Install Microsoft SQL Server tools
RUN export DEBIAN_FRONTEND=noninteractive && \
    curl -sSL https://packages.microsoft.com/keys/microsoft.asc | gpg --dearmor -o /usr/share/keyrings/microsoft-prod.gpg && \
    echo "deb [arch=${TARGETARCH} signed-by=/usr/share/keyrings/microsoft-prod.gpg] https://packages.microsoft.com/ubuntu/$(lsb_release -rs)/prod $(lsb_release -cs) main" > /etc/apt/sources.list.d/mssql-release.list && \
    apt-get update && \
    ACCEPT_EULA=Y apt-get install -y mssql-tools18 unixodbc-dev && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Add mssql-tools to PATH
ENV PATH="/opt/mssql-tools18/bin:$PATH"


# Install kando
COPY --from=tools_image /usr/local/bin/kando /usr/local/bin/kando

# Use Docker's TARGETARCH for kubectl architecture
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/${kubectl_version}/bin/linux/${TARGETARCH}/kubectl \
    && chmod +x ./kubectl \
    && mv ./kubectl /usr/local/bin/kubectl

CMD ["/usr/bin/tail", "-f", "/dev/null"]
