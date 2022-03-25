# SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
#
# SPDX-License-Identifier: Apache-2.0

export CGO_ENABLED=1
export GO111MODULE=on

.PHONY: build

ONOS_CONFIG_MODEL_VERSION ?= latest

PHONY:build

build-tools:=$(shell if [ ! -d "./build/build-tools" ]; then cd build && git clone https://github.com/onosproject/build-tools.git; fi)
include ./build/build-tools/make/onf-common.mk

build: # @HELP build all libraries
build: linters license

test: # @HELP run the unit tests and source code validation producing a golang style report
test: build deps license linters
	go test -race github.com/onosproject/onos-config-model/...

jenkins-test: # @HELP run the unit tests and source code validation producing a junit style report for Jenkins
jenkins-test: build deps license linters
	TEST_PACKAGES=github.com/onosproject/onos-config-model/pkg/... ./build/build-tools/build/jenkins/make-unit

serve: # @HELP start the registry server
serve:
	docker run -it \
		-v `pwd`/examples:/onos-config-model/models \
		-v `pwd`/build/plugins:/onos-config-model/build \
		-p 5151:5151 \
		onosproject/config-model-registry:go-${ONOS_CONFIG_MODEL_VERSION} \
		--registry-path /onos-config-model/models \
		--build-path /onos-config-model/build

images:
	./build/bin/build-images ${ONOS_CONFIG_MODEL_VERSION}

kind: # @HELP build Docker images and add them to the currently configured kind cluster
kind: images
	@if [ "`kind get clusters`" = '' ]; then echo "no kind cluster found" && exit 1; fi
	./build/bin/load-images ${ONOS_CONFIG_MODEL_VERSION}

push: images
	./build/bin/push-images ${ONOS_CONFIG_MODEL_VERSION}

publish: # @HELP publish version on github and dockerhub
	./build/build-tools/publish-version ${VERSION} onosproject/config-model-init onosproject/config-model-registry onosproject/config-model-build

jenkins-publish: images # @HELP Jenkins calls this to publish artifacts
	./build/bin/push-images ${ONOS_CONFIG_MODEL_VERSION}
	./build/build-tools/release-merge-commit
	./build/build-tools/build/docs/push-docs

clean:: # @HELP remove all the build artifacts
	@rm -r `pwd`/models
	@rm -r `pwd`/build/plugins

