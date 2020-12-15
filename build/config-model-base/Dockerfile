FROM golang:alpine

ENV GO111MODULE=on

RUN apk upgrade --update --no-cache && \
    apk add --no-cache gcc musl-dev && \
    go get -u github.com/openconfig/ygot/generator

COPY go.mod go.sum /onos-config-model/

WORKDIR /onos-config-model

RUN go mod download -x

COPY logging.yaml /onos-config-model/
COPY api /onos-config-model/api
COPY cmd /onos-config-model/cmd
COPY pkg /onos-config-model/pkg
