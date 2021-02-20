module github.com/onosproject/onos-config-model/{{ .Model.Name }}_{{ .Model.Version | replace "." "_" }}

go 1.14

require (
    github.com/onosproject/onos-config-model v0.1.1
)
