package configmodel

import (
    "errors"
	"github.com/openconfig/ygot/ygot"

	"github.com/onosproject/onos-config-model-go/pkg/model"
)

// Validator defines the validator for {{ .Model.Name }} {{ .Model.Version }}
type Validator struct{}

func (v Validator) Validate(ygotModel *ygot.ValidatedGoStruct, opts ...ygot.ValidationOption) error {
	deviceDeref := *ygotModel
	device, ok := deviceDeref.(*Device)
	if !ok {
		return errors.New("unable to convert model")
	}
	return device.Validate()
}

var _ configmodel.Validator = Validator{}
