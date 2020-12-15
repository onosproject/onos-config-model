FROM onosproject/golang-build:v0.6.3

COPY go.mod go.sum /onos-config-model/

WORKDIR /onos-config-model

RUN go mod download -x

COPY logging.yaml /onos-config-model/
COPY api /onos-config-model/api
COPY cmd /onos-config-model/cmd
COPY pkg /onos-config-model/pkg
