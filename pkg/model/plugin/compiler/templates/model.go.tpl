package configmodel

import (
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"

	"github.com/onosproject/onos-config-model/pkg/model"
)

const (
    modelName    configmodel.Name    = {{ .Model.Name | quote }}
    modelVersion configmodel.Version = {{ .Model.Version | quote }}
)

var modelData = []*gnmi.ModelData{
    {{- range .Model.Modules }}
	{Name: {{ .Name | quote }}, Organization: {{ .Organization | quote }}, Version: {{ .Version | quote }}},
	{Name: {{ .Name | quote }}, Organization: {{ .Organization | quote }}, Version: {{ .Version | quote }}},
	{{- end }}
}

var ModelInfo = configmodel.ModelInfo{
    Name: configmodel.Name({{ .Model.Name | quote }}),
    Version: configmodel.Version({{ .Model.Version | quote }}),
}

// ConfigModel defines the config model for {{ .Model.Name }} {{ .Model.Version }}
type ConfigModel struct{}

func (m ConfigModel) Info() configmodel.ModelInfo {
    return ModelInfo
}

func (m ConfigModel) Data() []*gnmi.ModelData {
    return modelData
}

func (m ConfigModel) Schema() (map[string]*yang.Entry, error) {
	return UnzipSchema()
}

func (m ConfigModel) GetStateMode() configmodel.GetStateMode {
    return configmodel.GetStateNone
}

func (m ConfigModel) Unmarshaller() configmodel.Unmarshaller {
    return Unmarshaller{}
}

func (m ConfigModel) Validator() configmodel.Validator {
    return Validator{}
}

var _ configmodel.ConfigModel = ConfigModel{}
