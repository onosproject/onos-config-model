ARG GOLANG_BUILD_VERSION=stable

FROM onosproject/golang-build:$GOLANG_BUILD_VERSION
ENV GO111MODULE=on
COPY . /root/config-model
WORKDIR /root/config-model
RUN go mod download

ENTRYPOINT ["go", "run", "github.com/onosproject/onos-config-model-go/cmd/config-model", "registry", "serve"]
