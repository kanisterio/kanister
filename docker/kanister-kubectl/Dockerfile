ARG TOOLS_IMAGE

FROM $TOOLS_IMAGE
ENV kubectl_version="v1.18.0"

LABEL name="kanister-kubectl" \
    vendor="Kanister" \
    version="${kubectl_version}" \
    summary="Kanster tools with kubectl" \
    maintainer="Tom Manville<tom@kasten.io>"

RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/${kubectl_version}/bin/linux/amd64/kubectl \
    && chmod +x ./kubectl \
    && mv ./kubectl /usr/local/bin/kubectl
