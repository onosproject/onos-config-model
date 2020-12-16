ARG GOLANG_BUILD_VERSION=latest
ARG CONFIG_MODEL_VERSION=latest

FROM onosproject/config-model-base:go-${CONFIG_MODEL_VERSION}-build-${GOLANG_BUILD_VERSION}

ENTRYPOINT ["go", "run", "github.com/onosproject/onos-config-model-go/cmd/config-model", "registry", "serve"]
