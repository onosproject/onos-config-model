# SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
#
# SPDX-License-Identifier: Apache-2.0

ARG CONFIG_MODEL_VERSION=latest

FROM onosproject/config-model-base:${CONFIG_MODEL_VERSION}

ENTRYPOINT ["go", "run", "github.com/onosproject/onos-config-model/cmd/config-model", "init"]
