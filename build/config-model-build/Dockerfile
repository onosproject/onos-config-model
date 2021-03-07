FROM golang:1.14-alpine3.13

RUN apk upgrade --update --no-cache && apk add --update make gcc musl-dev

RUN mkdir /build

WORKDIR /build

ENTRYPOINT ["make"]