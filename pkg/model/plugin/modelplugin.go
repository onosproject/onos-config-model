// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package modelplugin

import (
	"github.com/onosproject/onos-config-model/pkg/model"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"path/filepath"
	"plugin"
)

const pluginSymbol = "ConfigModelPlugin"

// ConfigModelPlugin provides a config model
type ConfigModelPlugin interface {
	// Model returns the config model
	Model() configmodel.ConfigModel
}

// Load loads the plugin at the given path
func Load(path string) (ConfigModelPlugin, error) {
	module, err := plugin.Open(path)
	if err != nil {
		return nil, err
	}
	symbol, err := module.Lookup(pluginSymbol)
	if err != nil {
		return nil, err
	}
	plugin, ok := symbol.(ConfigModelPlugin)
	if !ok {
		return nil, errors.NewInvalid("symbol loaded from module %s is not a %s", filepath.Base(path), pluginSymbol)
	}
	return plugin, nil
}
