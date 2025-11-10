ARG TOOLS_IMAGE
FROM ${TOOLS_IMAGE} AS TOOLS_IMAGE

FROM mcr.microsoft.com/mssql-tools
ENV kubectl_version="v1.18.0"
ENV GLIBC_VERSION="2.35"

# Install kando
COPY --from=TOOLS_IMAGE /usr/local/bin/kando /usr/local/bin/kando

# Install build deps
RUN apt-get update && apt-get install -y build-essential wget

# Build and install glibc ${GLIBC_VERSION} in /opt/glibc-${GLIBC_VERSION}
RUN wget http://ftp.gnu.org/gnu/libc/glibc-${GLIBC_VERSION}.tar.gz && \
    tar -xvf glibc-${GLIBC_VERSION}.tar.gz && \
    cd glibc-${GLIBC_VERSION} && mkdir build && cd build && \
    ../configure --prefix=/opt/glibc-${GLIBC_VERSION} && \
    make -j4 && make install

# Clean up build dependencies and source files
RUN apt-get remove --purge -y build-essential wget && apt-get clean && rm -rf /var/lib/apt/lists/* glibc-${GLIBC_VERSION}* glibc-build

RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/${kubectl_version}/bin/linux/amd64/kubectl \
    && chmod +x ./kubectl \
    && mv ./kubectl /usr/local/bin/kubectl

CMD [ "/usr/bin/tail", "-f", "/dev/null" ]
