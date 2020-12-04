package model

import (
	_ "github.com/golang/protobuf/proto"
	_ "github.com/openconfig/gnmi/proto/gnmi"
	_ "github.com/openconfig/goyang/pkg/yang"
	_ "github.com/openconfig/ygot/genutil"
	_ "github.com/openconfig/ygot/ygen"
	_ "github.com/openconfig/ygot/ygot"
	_ "github.com/openconfig/ygot/ytypes"

	"github.com/onosproject/onos-config-model-go/pkg/model"
)

// ConfigPlugin defines the model plugin for {{ .Model.Name }} {{ .Model.Version }}
type ConfigPlugin struct{}

func (p ConfigPlugin) Model() model.ConfigModel {
    return ConfigModel{}
}

var _ model.ConfigPlugin = ConfigPlugin{}
