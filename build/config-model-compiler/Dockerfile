ARG CONFIG_MODEL_VERSION=latest

FROM onosproject/config-model-base:${CONFIG_MODEL_VERSION}

ENTRYPOINT ["go", "run", "github.com/onosproject/onos-config-model/cmd/config-model", "compile"]
