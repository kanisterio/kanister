FROM node:14-alpine
RUN npm i -g license-extractor
RUN mkdir /repo

WORKDIR /repo

ENTRYPOINT ["licext"]
