FROM golang:1.14-alpine3.13

RUN apk upgrade --update --no-cache

RUN mkdir /build

WORKDIR /build

ENTRYPOINT ["make"]