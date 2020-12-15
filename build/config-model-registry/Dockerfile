ARG CONFIG_MODEL_BASE_VERSION=latest
FROM onosproject/config-model-base:go-$CONFIG_MODEL_BASE_VERSION

ENTRYPOINT ["go", "run", "github.com/onosproject/onos-config-model-go/cmd/config-model", "registry", "serve"]
