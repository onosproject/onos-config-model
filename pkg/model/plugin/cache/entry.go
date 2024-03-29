// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package plugincache

import (
	"context"
	"fmt"
	configmodel "github.com/onosproject/onos-config-model/pkg/model"
	modelplugin "github.com/onosproject/onos-config-model/pkg/model/plugin"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"os"
	"path/filepath"
)

func newPluginEntry(path string, name configmodel.Name, version configmodel.Version) *PluginEntry {
	return &PluginEntry{
		Path: filepath.Join(path, fmt.Sprintf("%s-%s.so", name, version)),
		lock: newPluginLock(filepath.Join(path, fmt.Sprintf("%s-%s.lock", name, version))),
	}
}

// PluginEntry is an entry for a plugin in the cache
type PluginEntry struct {
	Path string
	lock *pluginLock
}

// Lock acquires a write lock on the cache
func (e *PluginEntry) Lock(ctx context.Context) error {
	return e.lock.Lock(ctx)
}

// IsLocked checks whether the cache is write locked
func (e *PluginEntry) IsLocked() bool {
	return e.lock.IsLocked()
}

// Unlock releases a write lock from the cache
func (e *PluginEntry) Unlock(ctx context.Context) error {
	return e.lock.Unlock(ctx)
}

// RLock acquires a read lock on the cache
func (e *PluginEntry) RLock(ctx context.Context) error {
	return e.lock.RLock(ctx)
}

// IsRLocked checks whether the cache is read locked
func (e *PluginEntry) IsRLocked() bool {
	return e.lock.IsRLocked()
}

// RUnlock releases a read lock on the cache
func (e *PluginEntry) RUnlock(ctx context.Context) error {
	return e.lock.RUnlock(ctx)
}

// Cached returns whether the plugin is cached
func (e *PluginEntry) Cached() (bool, error) {
	if !e.IsRLocked() {
		return false, errors.NewConflict("cache is not locked")
	}
	if _, err := os.Stat(e.Path); !os.IsNotExist(err) {
		return true, nil
	}
	return false, nil
}

// Load loads the plugin from the cache
func (e *PluginEntry) Load() (modelplugin.ConfigModelPlugin, error) {
	if !e.IsRLocked() {
		return nil, errors.NewConflict("cache is not locked")
	}
	return modelplugin.Load(e.Path)
}
