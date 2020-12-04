package model

import (
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"

	"github.com/onosproject/onos-config-model-go/pkg/model"
)

const (
    modelName    model.Name    = {{ .Model.Name | quote }}
    modelVersion model.Version = {{ .Model.Version | quote }}
)

var modelData = []*gnmi.ModelData{
    {{- range .Model.Modules }}
	{Name: {{ .Name | quote }}, Organization: {{ .Organization | quote }}, Version: {{ .Version | quote }}},
	{Name: {{ .Name | quote }}, Organization: {{ .Organization | quote }}, Version: {{ .Version | quote }}},
	{{- end }}
}

var ConfigModelInfo = model.ConfigModelInfo{
    Name: model.Name({{ .Model.Name | quote }}),
    Version: model.Version({{ .Model.Version | quote }}),
}

// ConfigModel defines the config model for {{ .Model.Name }} {{ .Model.Version }}
type ConfigModel struct{}

func (m ConfigModel) Info() model.ConfigModelInfo {
    return ConfigModelInfo
}

func (m ConfigModel) Data() []*gnmi.ModelData {
    return modelData
}

func (m ConfigModel) Schema() (map[string]*yang.Entry, error) {
	return UnzipSchema()
}

func (m ConfigModel) GetStateMode() model.GetStateMode {
    return model.GetStateNone
}

func (m ConfigModel) Unmarshaller() model.ConfigModelUnmarshaller {
    return ConfigModelUnmarshaller{}
}

func (m ConfigModel) Validator() model.ConfigModelValidator {
    return ConfigModelValidator{}
}

var _ model.ConfigModel = ConfigModel{}
