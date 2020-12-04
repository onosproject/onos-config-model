export CGO_ENABLED=0
export GO111MODULE=on

.PHONY: build

ONOS_CONFIG_MODELS_VERSION ?= latest
ONOS_PROTOC_VERSION := v0.6.7
GOLANG_BUILD_VERSION  := v0.6.3

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
		-v `pwd`/plugins:/root/plugins \
		onosproject/config-agent:latest \
		plugin compile \
		--name test \
		--version 1.0.0 \
		--module test@2020-11-18=/root/plugins/test/test@2020-11-18.yang \
		--build-path /root/build/test
		--output-path /root/plugins

serve: # @HELP start the repo server
serve:
	docker run -it \
		-v `pwd`/models:/root/models \
		-v `pwd`/build/plugins:/root/build \
		-p 5150:5150 \
		onosproject/config-agent:latest \
		repo serve \
		--repo-path /root/models \
		--build-path /root/build

images: # @HELP build Docker images
images:
	docker build . -f build/config-agent/Dockerfile \
		--build-arg GOLANG_BUILD_VERSION=${GOLANG_BUILD_VERSION} \
		-t onosproject/config-agent:${ONOS_CONFIG_MODELS_VERSION}

kind: # @HELP build Docker images and add them to the currently configured kind cluster
kind: images
	@if [ "`kind get clusters`" = '' ]; then echo "no kind cluster found" && exit 1; fi
	kind load docker-image onosproject/config-agent:${ONOS_CONFIG_MODELS_VERSION}

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
