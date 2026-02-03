# We get tools from tools image
# Tools are not up to date in debian repos
ARG TOOLS_IMAGE
FROM ${TOOLS_IMAGE} AS tools_image

FROM debian:bookworm@sha256:23f8d499dcf5559a92ef814c309d747268420cf5e5e60f067f495d509f0aeb93
COPY --from=tools_image /usr/local/bin/restic /usr/local/bin/restic
COPY --from=tools_image /usr/local/bin/kopia /usr/local/bin/kopia
# Install kando
COPY --from=tools_image /usr/local/bin/kando /usr/local/bin/kando


RUN apt update
RUN apt install -y npm bash curl libcap2-bin
RUN curl -fsSL https://deb.nodesource.com/setup_current.x | bash - && \
 apt-get install -y nodejs
RUN npm install -g npm
RUN npm install elasticdump -g

RUN setcap cap_chown,cap_fowner,cap_dac_override+iep /usr/local/bin/kopia

CMD [ "/usr/bin/tail", "-f", "/dev/null" ]
