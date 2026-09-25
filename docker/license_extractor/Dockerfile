FROM node:26-alpine@sha256:2c45bdcbf63561a54da9549612084b43ca309854a4110c87857d609ddeb61c9e
RUN npm i -g license-extractor
RUN mkdir /repo

WORKDIR /repo

ENTRYPOINT ["licext"]
