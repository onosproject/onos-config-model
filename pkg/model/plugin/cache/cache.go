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
	configmodel "github.com/onosproject/onos-config-model/pkg/model"
	modelplugin "github.com/onosproject/onos-config-model/pkg/model/plugin"
	pluginmodule "github.com/onosproject/onos-config-model/pkg/model/plugin/module"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"os"
	"path/filepath"
	"sync"
	"syscall"
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
	path     string
	rlocked  bool
	wlocked  bool
	fh       *os.File
	mu       sync.RWMutex
}

// Lock acquires a write lock on the cache
func (c *PluginCache) Lock(ctx context.Context) error {
	locked, err := c.lock(ctx, &c.wlocked, syscall.LOCK_EX)
	if err != nil {
		err = errors.NewInternal(err.Error())
		log.Error(err)
		return err
	} else if !locked {
		err = errors.NewConflict("failed to acquire cache lock")
		log.Error(err)
		return err
	}
	return nil
}

// IsLocked checks whether the cache is write locked
func (c *PluginCache) IsLocked() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.wlocked
}

// Unlock releases a write lock from the cache
func (c *PluginCache) Unlock(ctx context.Context) error {
	if err := c.unlock(); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

// RLock acquires a read lock on the cache
func (c *PluginCache) RLock(ctx context.Context) error {
	locked, err := c.lock(ctx, &c.rlocked, syscall.LOCK_SH)
	if err != nil {
		err = errors.NewInternal(err.Error())
		log.Error(err)
		return err
	} else if !locked {
		err = errors.NewConflict("failed to acquire cache lock")
		log.Error(err)
		return err
	}
	return nil
}

// IsRLocked checks whether the cache is read locked
func (c *PluginCache) IsRLocked() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.wlocked || c.rlocked
}

// RUnlock releases a read lock on the cache
func (c *PluginCache) RUnlock(ctx context.Context) error {
	if err := c.unlock(); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

// lock attempts to acquire a file lock
func (c *PluginCache) lock(ctx context.Context, locked *bool, flag int) (bool, error) {
	for {
		if ok, err := c.tryLock(locked, flag); ok || err != nil {
			return ok, err
		}
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(lockAttemptDelay):
			// try again
		}
	}
}

func (c *PluginCache) tryLock(locked *bool, flag int) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if *locked {
		return true, nil
	}

	if c.fh == nil {
		if err := c.openFH(); err != nil {
			return false, err
		}
		defer c.ensureFhState()
	}

	err := syscall.Flock(int(c.fh.Fd()), flag|syscall.LOCK_NB)
	switch err {
	case syscall.EWOULDBLOCK:
		return false, nil
	case nil:
		*locked = true
		return true, nil
	}
	return false, err
}

func (c *PluginCache) unlock() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if (!c.wlocked && !c.rlocked) || c.fh == nil {
		return nil
	}

	if err := syscall.Flock(int(c.fh.Fd()), syscall.LOCK_UN); err != nil {
		return err
	}

	c.fh.Close()

	c.wlocked = false
	c.rlocked = false
	c.fh = nil
	return nil
}

func (c *PluginCache) openFH() error {
	if c.path == "" {
		cacheDir, err := c.getModCache()
		if err != nil {
			return err
		}

		if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
			if err := os.MkdirAll(cacheDir, os.ModePerm); err != nil {
				return err
			}
		}
		c.path = filepath.Join(cacheDir, lockFileName)
	}

	fh, err := os.OpenFile(c.path, os.O_CREATE|os.O_RDONLY, os.FileMode(0666))
	if err != nil {
		return err
	}
	c.fh = fh
	return nil
}

func (c *PluginCache) ensureFhState() {
	if !c.wlocked && !c.rlocked && c.fh != nil {
		c.fh.Close()
		c.fh = nil
	}
}

// getModCache gets the cache directory for the module target
func (c *PluginCache) getModCache() (string, error) {
	_, hash, err := c.resolver.Resolve()
	if err != nil {
		return "", err
	}
	return filepath.Join(c.Config.Path, base64.URLEncoding.EncodeToString(hash)), nil
}

// GetPath gets the path of the given plugin
func (c *PluginCache) GetPath(name configmodel.Name, version configmodel.Version) (string, error) {
	cacheDir, err := c.getModCache()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, fmt.Sprintf("%s-%s.so", name, version)), nil
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
