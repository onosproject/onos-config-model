export CGO_ENABLED=0
export GO111MODULE=on

.PHONY: build

ONOS_CONFIG_MODEL_VERSION ?= latest
ONOS_PROTOC_VERSION := v0.6.7
GOLANG_BUILD_VERSIONS  := v0.6.3 v0.6.6
DEFAULT_GOLANG_BUILD_VERSION := v0.6.3

linters: # @HELP examines Go source code and reports coding problems
	golangci-lint run --timeout 30m

license_check: # @HELP examine and ensure license headers exist
	@if [ ! -d "../build-tools" ]; then cd .. && git clone https://github.com/onosproject/build-tools.git; fi
	./../build-tools/licensing/boilerplate.py -v --rootdir=${CURDIR}

gofmt: # @HELP run the Go format validation
	bash -c "diff -u <(echo -n) <(gofmt -d pkg/)"

PHONY:build
build: # @HELP build all libraries
build: linters license_check gofmt

protos: # @HELP compile the protobuf files (using protoc-go Docker)
	docker run -it -v `pwd`:/go/src/github.com/onosproject/onos-config-model-go \
		-w /go/src/github.com/onosproject/onos-config-model-go \
		--entrypoint build/bin/compile-protos.sh \
		onosproject/protoc-go:${ONOS_PROTOC_VERSION}

compile-plugins: # @HELP compile standard plugins
compile-plugins:
	docker run \
		-v `pwd`/examples:/onos-config-model/plugins \
		-v `pwd`/build/_output:/onos-config-model/build \
		onosproject/config-model-compiler:go-${ONOS_CONFIG_MODEL_VERSION} \
		--name test \
		--version 1.0.0 \
		--module test@2020-11-18=/onos-config-model/plugins/test@2020-11-18.yang \
		--target github.com/onosproject/onos-config \
		--replace github.com/kuujo/onos-config@f4d3d81 \
		--build-path /onos-config-model/build \
		--output-path /onos-config-model/plugins

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
	./build/bin/build-images ${ONOS_CONFIG_MODEL_VERSION} ${DEFAULT_GOLANG_BUILD_VERSION} ${GOLANG_BUILD_VERSIONS}

kind: # @HELP build Docker images and add them to the currently configured kind cluster
kind: images
	@if [ "`kind get clusters`" = '' ]; then echo "no kind cluster found" && exit 1; fi
	./build/bin/load-images ${ONOS_CONFIG_MODEL_VERSION} ${DEFAULT_GOLANG_BUILD_VERSION} ${GOLANG_BUILD_VERSIONS}

clean: # @HELP remove all the build artifacts
	@rm -r `pwd`/models
	@rm -r `pwd`/build/plugins

help:
	@grep -E '^.*: *# *@HELP' $(MAKEFILE_LIST) \
    | sort \
    | awk ' \
        BEGIN {FS = ": *# *@HELP"}; \
        {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}; \
    '
