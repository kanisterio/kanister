FROM node:26-alpine@sha256:143494b1da2945f061539253adc65e4f1569ddf07da2d384c022c791a9d90a4a
RUN npm i -g license-extractor
RUN mkdir /repo

WORKDIR /repo

ENTRYPOINT ["licext"]
