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
	"context"
	"fmt"
	"github.com/gofrs/flock"
	configmodel "github.com/onosproject/onos-config-model/pkg/model"
	modelplugin "github.com/onosproject/onos-config-model/pkg/model/plugin"
	pluginmodule "github.com/onosproject/onos-config-model/pkg/model/plugin/module"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"os"
	"path/filepath"
	"time"
)

var log = logging.GetLogger("config-model", "plugin", "cache")

const (
	defaultPath      = "/etc/onos/plugins"
	lockFileName     = "cache.lock"
	lockAttemptDelay = 5 * time.Second
)

// CacheConfig is a plugin cache configuration
type CacheConfig struct {
	Path string `yaml:"path" json:"path"`
}

// NewPluginCache creates a new plugin cache
func NewPluginCache(config CacheConfig, resolver *pluginmodule.Resolver) *PluginCache {
	if config.Path == "" {
		config.Path = defaultPath
	}
	return &PluginCache{
		Config:   config,
		resolver: resolver,
		lock:     flock.New(filepath.Join(config.Path, lockFileName)),
	}
}

// PluginCache is a model plugin cache
type PluginCache struct {
	Config   CacheConfig
	resolver *pluginmodule.Resolver
	lock     *flock.Flock
}

// Lock acquires a write lock on the cache
func (c *PluginCache) Lock() error {
	succeeded, err := c.lock.TryLockContext(context.Background(), lockAttemptDelay)
	if err != nil {
		return errors.NewInternal(err.Error())
	} else if !succeeded {
		return errors.NewConflict("failed to acquire cache lock")
	}
	return nil
}

// IsLocked checks whether the cache is write locked
func (c *PluginCache) IsLocked() bool {
	return c.lock.Locked()
}

// Unlock releases a write lock from the cache
func (c *PluginCache) Unlock() error {
	return c.lock.Unlock()
}

// RLock acquires a read lock on the cache
func (c *PluginCache) RLock() error {
	succeeded, err := c.lock.TryRLockContext(context.Background(), lockAttemptDelay)
	if err != nil {
		return errors.NewInternal(err.Error())
	} else if !succeeded {
		return errors.NewConflict("failed to acquire cache lock")
	}
	return nil
}

// IsRLocked checks whether the cache is read locked
func (c *PluginCache) IsRLocked() bool {
	return c.lock.Locked() || c.lock.RLocked()
}

// RUnlock releases a read lock on the cache
func (c *PluginCache) RUnlock() error {
	return c.lock.Unlock()
}

// GetPath gets the path of the given plugin
func (c *PluginCache) GetPath(name configmodel.Name, version configmodel.Version) (string, error) {
	_, hash, err := c.resolver.Resolve()
	if err != nil {
		return "", err
	}
	return filepath.Join(c.Config.Path, string(hash), fmt.Sprintf("%s-%s.so", name, version)), nil
}

// Cached returns whether the given plugin is cached
func (c *PluginCache) Cached(name configmodel.Name, version configmodel.Version) (bool, error) {
	if !c.IsRLocked() {
		return false, errors.NewConflict("cache is not locked")
	}
	path, err := c.GetPath(name, version)
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return true, nil
	}
	return false, nil
}

// Load loads a plugin from the cache
func (c *PluginCache) Load(name configmodel.Name, version configmodel.Version) (modelplugin.ConfigModelPlugin, error) {
	if !c.IsRLocked() {
		return nil, errors.NewConflict("cache is not locked")
	}
	path, err := c.GetPath(name, version)
	if err != nil {
		return nil, err
	}
	return modelplugin.Load(path)
}
