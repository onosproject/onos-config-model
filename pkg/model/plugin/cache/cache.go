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

package plugincache

import (
	"encoding/base64"
	"fmt"
	configmodel "github.com/onosproject/onos-config-model/pkg/model"
	pluginmodule "github.com/onosproject/onos-config-model/pkg/model/plugin/module"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var log = logging.GetLogger("config-model", "plugin", "cache")

const (
	defaultPath      = "/etc/onos/plugins"
	lockAttemptDelay = 5 * time.Second
)

// CacheConfig is a plugin cache configuration
type CacheConfig struct {
	Path string `yaml:"path" json:"path"`
}

// NewPluginCache creates a new plugin cache
func NewPluginCache(config CacheConfig, resolver *pluginmodule.Resolver) (*PluginCache, error) {
	if config.Path == "" {
		config.Path = defaultPath
	}

	_, hash, err := resolver.Resolve()
	if err != nil {
		return nil, err
	}

	config.Path = filepath.Join(config.Path, base64.RawURLEncoding.EncodeToString(hash))
	if _, err := os.Stat(config.Path); os.IsNotExist(err) {
		if err := os.MkdirAll(config.Path, os.ModePerm); err != nil {
			return nil, err
		}
	}
	return &PluginCache{
		Config:  config,
		entries: make(map[string]*PluginEntry),
	}, nil
}

// PluginCache is a model plugin cache
type PluginCache struct {
	Config  CacheConfig
	entries map[string]*PluginEntry
	mu      sync.RWMutex
}

// Entry returns the entry for the given plugin name+version
func (c *PluginCache) Entry(name configmodel.Name, version configmodel.Version) *PluginEntry {
	path := fmt.Sprintf("%s-%s", name, version)
	c.mu.RLock()
	entry, ok := c.entries[path]
	c.mu.RUnlock()
	if ok {
		return entry
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok = c.entries[path]
	if ok {
		return entry
	}

	entry = newPluginEntry(c.Config.Path, name, version)
	c.entries[path] = entry
	return entry
}
