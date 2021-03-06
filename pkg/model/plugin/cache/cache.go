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
	"encoding/base64"
	"fmt"
	"github.com/gofrs/flock"
	configmodel "github.com/onosproject/onos-config-model/pkg/model"
	modelplugin "github.com/onosproject/onos-config-model/pkg/model/plugin"
	pluginmodule "github.com/onosproject/onos-config-model/pkg/model/plugin/module"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"os"
	"path/filepath"
	"sync"
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
	}
}

// PluginCache is a model plugin cache
type PluginCache struct {
	Config   CacheConfig
	resolver *pluginmodule.Resolver
	lock     *flock.Flock
	mu       sync.RWMutex
}

// Lock acquires a write lock on the cache
func (c *PluginCache) Lock() error {
	lock, err := c.getLock()
	if err != nil {
		return err
	}
	succeeded, err := lock.TryLockContext(context.Background(), lockAttemptDelay)
	if err != nil {
		return errors.NewInternal(err.Error())
	} else if !succeeded {
		return errors.NewConflict("failed to acquire cache lock")
	}
	return nil
}

// IsLocked checks whether the cache is write locked
func (c *PluginCache) IsLocked() bool {
	c.mu.RLock()
	lock := c.lock
	c.mu.RUnlock()
	return lock != nil && lock.Locked()
}

// Unlock releases a write lock from the cache
func (c *PluginCache) Unlock() error {
	lock, err := c.getLock()
	if err != nil {
		return err
	}
	return lock.Unlock()
}

// RLock acquires a read lock on the cache
func (c *PluginCache) RLock() error {
	lock, err := c.getLock()
	if err != nil {
		return err
	}
	succeeded, err := lock.TryRLockContext(context.Background(), lockAttemptDelay)
	if err != nil {
		return errors.NewInternal(err.Error())
	} else if !succeeded {
		return errors.NewConflict("failed to acquire cache lock")
	}
	return nil
}

// IsRLocked checks whether the cache is read locked
func (c *PluginCache) IsRLocked() bool {
	c.mu.RLock()
	lock := c.lock
	c.mu.RUnlock()
	return lock != nil && (lock.Locked() || lock.RLocked())
}

// RUnlock releases a read lock on the cache
func (c *PluginCache) RUnlock() error {
	lock, err := c.getLock()
	if err != nil {
		return err
	}
	return lock.Unlock()
}

// getLock gets the cache lock for the resolved module target, initializing the lock with the
// correct permissions (0666) if necessary
func (c *PluginCache) getLock() (*flock.Flock, error) {
	c.mu.RLock()
	lock := c.lock
	c.mu.RUnlock()
	if lock != nil {
		return lock, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	dir, err := c.getDir()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return nil, err
		}
	}

	file := filepath.Join(dir, lockFileName)
	if _, err = os.Create(file); err != nil {
		return nil, err
	}

	lock = flock.New(file)
	c.lock = lock
	return lock, nil
}

// getDir gets the cache directory for the module target
func (c *PluginCache) getDir() (string, error) {
	_, hash, err := c.resolver.Resolve()
	if err != nil {
		return "", err
	}
	return filepath.Join(c.Config.Path, base64.URLEncoding.EncodeToString(hash)), nil
}

// GetPath gets the path of the given plugin
func (c *PluginCache) GetPath(name configmodel.Name, version configmodel.Version) (string, error) {
	dir, err := c.getDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fmt.Sprintf("%s-%s.so", name, version)), nil
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

func openFH(file string) (*os.File, error) {
	dir := filepath.Dir(file)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return nil, err
		}
	}
	return os.OpenFile(file, os.O_CREATE|os.O_RDWR, os.FileMode(0666))
}
