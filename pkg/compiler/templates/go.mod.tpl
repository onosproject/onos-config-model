module github.com/onosproject/onos-config-model-go/{{ .Model.Name }}_{{ .Model.Version | replace "." "_" }}

go 1.14

require (
    github.com/onosproject/onos-config-model-go v1.0.0
)

replace github.com/onosproject/onos-config-model-go => {{ .Compiler.Root }}
