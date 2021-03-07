FROM golang:1.14-alpine3.13

RUN apk upgrade --update --no-cache

RUN export GO111MODULE=on && \
    go get -u github.com/golang/protobuf/protoc-gen-go && \
    go get -u google.golang.org/protobuf/proto && \
    go get -u golang.org/x/lint/golint && \
    go get -u github.com/fzipp/gocyclo && \
    go get -u github.com/client9/misspell/cmd/misspell && \
    go get -u github.com/gordonklaus/ineffassign && \
    go get -u github.com/openconfig/ygot/generator

RUN mkdir /build

WORKDIR /build

ENTRYPOINT ["make"]