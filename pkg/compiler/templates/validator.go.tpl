package model

import (
    "errors"
	"github.com/openconfig/ygot/ygot"

	"github.com/onosproject/onos-config-model-go/pkg/model"
)

// ConfigModelValidator defines the validator for {{ .Model.Name }} {{ .Model.Version }}
type ConfigModelValidator struct{}

func (v ConfigModelValidator) Validate(ygotModel *ygot.ValidatedGoStruct, opts ...ygot.ValidationOption) error {
	deviceDeref := *ygotModel
	device, ok := deviceDeref.(*Device)
	if !ok {
		return errors.New("unable to convert model")
	}
	return device.Validate()
}

var _ model.ConfigModelValidator = ConfigModelValidator{}
