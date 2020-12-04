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

package model

import (
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"path/filepath"
	"plugin"
)

const pluginSymbol = "ConfigPlugin"

// ConfigPlugin provides a config model
type ConfigPlugin interface {
	// Model returns the config model
	Model() ConfigModel
}

// Load loads the plugin at the given path
func Load(path string) (ConfigModel, error) {
	module, err := plugin.Open(path)
	if err != nil {
		return nil, err
	}
	symbol, err := module.Lookup(pluginSymbol)
	if err != nil {
		return nil, err
	}
	plugin, ok := symbol.(ConfigPlugin)
	if !ok {
		return nil, errors.NewInvalid("symbol loaded from module %s is not a ConfigPlugin", filepath.Base(path))
	}
	return plugin.Model(), nil
}
