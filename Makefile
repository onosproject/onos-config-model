export CGO_ENABLED=1
export GO111MODULE=on

.PHONY: build

ONOS_CONFIG_MODEL_VERSION ?= latest

PHONY:build
build: # @HELP build all libraries
build: linters license_check gofmt

test: # @HELP run the unit tests and source code validation producing a golang style report
test: build deps license_check linters
	go test -race github.com/onosproject/onos-config-model/...

jenkins-test: build-tools # @HELP run the unit tests and source code validation producing a junit style report for Jenkins
jenkins-test: build deps license_check linters
	TEST_PACKAGES=github.com/onosproject/onos-config-model/pkg/... ./../build-tools/build/jenkins/make-unit

deps: # @HELP ensure that the required dependencies are in place
	go build -v ./...
	bash -c "diff -u <(echo -n) <(git diff go.mod)"
	bash -c "diff -u <(echo -n) <(git diff go.sum)"

linters: golang-ci # @HELP examines Go source code and reports coding problems
	golangci-lint run --timeout 5m

build-tools: # @HELP install the ONOS build tools if needed
	@if [ ! -d "../build-tools" ]; then cd .. && git clone https://github.com/onosproject/build-tools.git; fi

jenkins-tools: # @HELP installs tooling needed for Jenkins
	cd .. && go get -u github.com/jstemmer/go-junit-report && go get github.com/t-yuki/gocover-cobertura

golang-ci: # @HELP install golang-ci if not present
	golangci-lint --version || curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b `go env GOPATH`/bin v1.36.0

license_check: build-tools # @HELP examine and ensure license headers exist
	./../build-tools/licensing/boilerplate.py -v --rootdir=${CURDIR}

gofmt: # @HELP run the Go format validation
	bash -c "diff -u <(echo -n) <(gofmt -d pkg/)"

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
	./build/bin/build-images ${ONOS_CONFIG_MODEL_VERSION}

kind: # @HELP build Docker images and add them to the currently configured kind cluster
kind: images
	@if [ "`kind get clusters`" = '' ]; then echo "no kind cluster found" && exit 1; fi
	./build/bin/load-images ${ONOS_CONFIG_MODEL_VERSION}

push: images
	./build/bin/push-images ${ONOS_CONFIG_MODEL_VERSION}

publish: # @HELP publish version on github and dockerhub
	./../build-tools/publish-version ${VERSION} onosproject/config-model-init onosproject/config-model-compiler onosproject/config-model-registry

jenkins-publish: build-tools jenkins-tools # @HELP Jenkins calls this to publish artifacts
	./build/bin/push-images ${ONOS_CONFIG_MODEL_VERSION}
	../build-tools/release-merge-commit

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
