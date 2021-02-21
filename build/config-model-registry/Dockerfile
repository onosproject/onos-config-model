ARG GOLANG_BUILD_VERSION=latest
ARG CONFIG_MODEL_VERSION=latest

FROM onosproject/config-model-base:${CONFIG_MODEL_VERSION}-golang-build-${GOLANG_BUILD_VERSION}

ENTRYPOINT ["go", "run", "github.com/onosproject/onos-config-model/cmd/config-model", "registry", "serve"]
