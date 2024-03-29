// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package modelregistry

import (
	"github.com/onosproject/onos-config-model/pkg/model"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestRegistry(t *testing.T) {
	dir, err := os.Getwd()
	assert.NoError(t, err)
	config := Config{
		Path: dir,
	}
	registry := NewConfigModelRegistry(config)

	_, err = registry.GetModel("foo", "1.0.0")
	assert.Error(t, err)
	assert.True(t, errors.IsNotFound(err))

	models, err := registry.ListModels()
	assert.NoError(t, err)
	assert.Len(t, models, 0)

	model := configmodel.ModelInfo{
		Name:    "foo",
		Version: "1.0.0",
		Modules: []configmodel.ModuleInfo{
			{
				Name:         "bar",
				Organization: "ONF",
				Revision:     "0.1.0",
				File:         "bar",
			},
		},
		Plugin: configmodel.PluginInfo{
			Name:    "foo",
			Version: "1.0.0",
		},
	}
	err = registry.AddModel(model)
	assert.NoError(t, err)

	model, err = registry.GetModel("foo", "1.0.0")
	assert.NoError(t, err)
	assert.Equal(t, configmodel.Name("foo"), model.Name)
	assert.Equal(t, configmodel.Version("1.0.0"), model.Version)

	models, err = registry.ListModels()
	assert.NoError(t, err)
	assert.Len(t, models, 1)

	err = registry.RemoveModel("foo", "1.0.0")
	assert.NoError(t, err)

	models, err = registry.ListModels()
	assert.NoError(t, err)
	assert.Len(t, models, 0)
}
