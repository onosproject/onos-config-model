# SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
#
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.17.8-alpine3.14

RUN apk upgrade --update --no-cache && apk add --update make gcc musl-dev git

RUN mkdir -p /build/deps && \
    cd /build/deps && \
    echo "module deps" > go.mod && \
    go get -u -d github.com/onosproject/onos-config && \
    go get -u -d github.com/openconfig/ygot/generator && \
    go mod download -x && \
    rm -r /build/deps

WORKDIR /build

ENTRYPOINT ["make"]
