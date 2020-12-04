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
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
)

// Name is a config model name
type Name string

// Version is a config model version
type Version string

// GetStateMode defines the Getstate handling
type GetStateMode int

const (
	// GetStateNone - device type does not support Operational State at all
	GetStateNone GetStateMode = iota
	// GetStateOpState - device returns all its op state attributes by querying
	// GetRequest_STATE and GetRequest_OPERATIONAL
	GetStateOpState
	// GetStateExplicitRoPaths - device returns all its op state attributes by querying
	// exactly what the ReadOnly paths from YANG - wildcards are handled by device
	GetStateExplicitRoPaths
	// GetStateExplicitRoPathsExpandWildcards - where there are wildcards in the
	// ReadOnly paths 2 calls have to be made - 1) to expand the wildcards in to
	// real paths (since the device doesn't do it) and 2) to query those expanded
	// wildcard paths - this is the Stratum 1.0.0 method
	GetStateExplicitRoPathsExpandWildcards
)

// ConfigModelInfo is config model info
type ConfigModelInfo struct {
	Name    Name               `json:"name"`
	Version Version            `json:"version"`
	Modules []ConfigModuleInfo `json:"modules"`
	Plugin  ConfigPluginInfo   `json:"plugin"`
}

// ConfigModuleInfo is a config module info
type ConfigModuleInfo struct {
	Name         Name    `json:"name"`
	Organization string  `json:"organization"`
	Version      Version `json:"version"`
	Data         []byte  `json:"data"`
}

// ConfigPluginInfo is config model plugin info
type ConfigPluginInfo struct {
	Name    Name    `json:"name"`
	Version Version `json:"version"`
	File    string  `json:"file"`
}

// ConfigModel is a configuration model data
type ConfigModel interface {
	// Info returns the config model info
	Info() ConfigModelInfo

	// Data returns the config model data
	Data() []*gnmi.ModelData

	// Schema returns the config model schema
	Schema() (map[string]*yang.Entry, error)

	// GetStateMode returns the get state mode
	GetStateMode() GetStateMode

	// ConfigModelUnmarshaller returns the config model unmarshaller
	Unmarshaller() ConfigModelUnmarshaller

	// ConfigModelValidator returns the config model validator
	Validator() ConfigModelValidator
}

// ConfigModelUnmarshaller is a config model unmarshaller
type ConfigModelUnmarshaller interface {
	// Unmarshal unmarshals the given config
	Unmarshal(bytes []byte) (*ygot.ValidatedGoStruct, error)
}

// ConfigModelValidator is a config model validator
type ConfigModelValidator interface {
	// Validate validates the given config struct
	Validate(model *ygot.ValidatedGoStruct, opts ...ygot.ValidationOption) error
}
