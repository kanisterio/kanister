FROM golang:1.14-alpine AS go

FROM node:14-alpine
RUN npm i -g license-extractor
RUN mkdir /repo

WORKDIR /repo

COPY --from=go /usr/local/go/ /usr/local/go/
RUN export PATH="/usr/local/go/bin:$PATH"

ENTRYPOINT ["licext"]
