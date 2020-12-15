package configmodel

import (
	"github.com/openconfig/ygot/ygot"

	"github.com/onosproject/onos-config-model-go/pkg/model"
)

// Unmarshaller defines the unmarshaller for {{ .Model.Name }} {{ .Model.Version }}
type Unmarshaller struct{}

func (u Unmarshaller) Unmarshal(jsonTree []byte) (*ygot.ValidatedGoStruct, error) {
    device := &Device{}
    vgs := ygot.ValidatedGoStruct(device)
    if err := Unmarshal([]byte(jsonTree), device); err != nil {
        return nil, err
    }
    return &vgs, nil
}

var _ configmodel.Unmarshaller = Unmarshaller{}
