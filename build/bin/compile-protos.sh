#!/bin/sh

proto_imports=".:${GOPATH}/src/github.com/gogo/protobuf/protobuf:${GOPATH}/src/github.com/gogo/protobuf:${GOPATH}/src"

protoc -I=$proto_imports --gogofaster_out=import_path=github.com/onosproject/onos-config-model-go/api/onos/configmodel,plugins=grpc:. api/onos/configmodel/*.proto