FROM node:25-alpine@sha256:f741690269ee7abb07675cb744f0b66ec117b482d89d9565a36f2360d5e2a3ef
RUN npm i -g license-extractor
RUN mkdir /repo

WORKDIR /repo

ENTRYPOINT ["licext"]
