# SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
#
# SPDX-License-Identifier: Apache-2.0

ARG CONFIG_MODEL_VERSION=latest

FROM onosproject/config-model-build:$CONFIG_MODEL_VERSION

COPY go.mod go.sum /onos-config-model/

WORKDIR /onos-config-model

RUN go mod download -x

COPY logging.yaml /onos-config-model/
COPY cmd /onos-config-model/cmd
COPY pkg /onos-config-model/pkg
