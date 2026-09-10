FROM node:26-alpine@sha256:ef24c5053d50fdc3e4e56eb4e7ddb7861874ab0fdc797046ba897581deb8e868
RUN npm i -g license-extractor
RUN mkdir /repo

WORKDIR /repo

ENTRYPOINT ["licext"]
