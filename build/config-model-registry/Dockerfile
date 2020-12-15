FROM golang:alpine

ENV GO111MODULE=on

RUN apk upgrade --update --no-cache && \
    apk add --no-cache gcc musl-dev && \
    go get -u github.com/openconfig/ygot/generator

COPY . /onos-config-model
WORKDIR /onos-config-model
RUN go mod download -x

ENTRYPOINT ["go", "run", "github.com/onosproject/onos-config-model-go/cmd/config-model", "registry", "serve"]
