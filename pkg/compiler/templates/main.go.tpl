package main

import (
	"github.com/onosproject/onos-config-model-go/{{ .Model.Name }}_{{ .Model.Version | replace "." "_" }}/model"
)

var ConfigPlugin model.ConfigPlugin
