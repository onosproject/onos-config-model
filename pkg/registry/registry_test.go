// Copyright 2020-present Open Networking Foundation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package registry

import (
	"github.com/onosproject/onos-config-model-go/pkg/model"
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
	registry := NewRegistry(config)

	models, err := registry.ListModels()
	assert.NoError(t, err)
	assert.Len(t, models, 0)

	modelInfo := model.ConfigModelInfo{
		Name:    "foo",
		Version: "1.0.0",
		Modules: []model.ConfigModuleInfo{
			{
				Name:         "bar",
				Organization: "ONF",
				Version:      "0.1.0",
				Data:         []byte("Hello world!"),
			},
		},
		Plugin: model.ConfigPluginInfo{
			Name:    "foo",
			Version: "1.0.0",
			File:    "foo@1.0.0.so",
		},
	}
	err = registry.AddModel(modelInfo)
	assert.NoError(t, err)

	models, err = registry.ListModels()
	assert.NoError(t, err)
	assert.Len(t, models, 1)

	err = registry.RemoveModel("foo", "1.0.0")
	assert.NoError(t, err)

	models, err = registry.ListModels()
	assert.NoError(t, err)
	assert.Len(t, models, 0)
}
