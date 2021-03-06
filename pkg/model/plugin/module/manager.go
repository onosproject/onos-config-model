// Copyright 2021-present Open Networking Foundation.
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

package module

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	_ "github.com/openconfig/gnmi/proto/gnmi" // gnmi
	_ "github.com/openconfig/goyang/pkg/yang" // yang
	_ "github.com/openconfig/ygot/genutil"    // genutil
	_ "github.com/openconfig/ygot/ygen"       // ygen
	_ "github.com/openconfig/ygot/ygot"       // ygot
	_ "github.com/openconfig/ygot/ytypes"     // ytypes
	"github.com/rogpeppe/go-internal/modfile"
	"github.com/rogpeppe/go-internal/module"
	_ "google.golang.org/protobuf/proto" // proto
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

var log = logging.GetLogger("config-model", "plugin", "module")

const (
	modFile  = "go.mod"
	hashFile = "mod.md5"
)

// Hash is a module hash
type Hash []byte

// ManagerConfig is a module manager configuration
type ManagerConfig struct {
	Path    string
	Target  string
	Replace string
}

// NewManager creates a new model plugin compiler
func NewManager(config ManagerConfig) *Manager {
	return &Manager{config}
}

// Manager is a module manager
type Manager struct {
	Config ManagerConfig
}

func (m *Manager) exec(dir string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GO111MODULE=on", "CGO_ENABLED=1")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (m *Manager) getGoEnv() (goEnv, error) {
	wd, err := os.Getwd()
	if err != nil {
		return goEnv{}, err
	}

	envJSON, err := m.exec(wd, "go", "env", "-json", "GOPATH", "GOMODCACHE")
	if err != nil {
		return goEnv{}, err
	}
	env := goEnv{}
	if err := json.Unmarshal([]byte(envJSON), &env); err != nil {
		return goEnv{}, err
	}
	return env, nil
}

func (m *Manager) getGoModCacheDir() (string, error) {
	env, err := m.getGoEnv()
	if err != nil {
		return "", err
	}
	modCache := env.GOMODCACHE
	if modCache == "" {
		// For Go 1.14 and older.
		return filepath.Join(env.GOPATH, "pkg", "mod"), nil
	}
	return modCache, nil
}

func (m *Manager) FetchMod() (*modfile.File, Hash, error) {
	modPath := m.getModPath()
	modBytes, modErr := ioutil.ReadFile(modPath)
	hashPath := m.getHashPath()
	hashBytes, hashErr := ioutil.ReadFile(hashPath)
	if modErr != nil || hashErr != nil {
		mod, hash, err := m.fetchMod()
		modBytes, err := mod.Format()
		if err != nil {
			return nil, nil, err
		}
		if err := ioutil.WriteFile(m.getModPath(), modBytes, 0666); err != nil {
			return nil, nil, err
		}
		if err := ioutil.WriteFile(m.getHashPath(), hash, 0666); err != nil {
			return nil, nil, err
		}
		return mod, hash, nil
	}
	modFile, err := modfile.Parse(modPath, modBytes, nil)
	if err != nil {
		return nil, nil, err
	}
	return modFile, hashBytes, nil
}

func (m *Manager) fetchMod() (*modfile.File, Hash, error) {
	target, replace := m.Config.Target, m.Config.Replace
	targetPath, _, _ := module.SplitPathVersion(target)

	log.Debugf("Fetching module '%s'", target)
	tmpDir, err := ioutil.TempDir("", "config-model")
	if err != nil {
		log.Error(err)
		return nil, nil, err
	}
	defer os.RemoveAll(tmpDir)

	// Generate a temporary module with which to pull the target module
	tmpMod := []byte("module m\n")
	if replace != "" {
		replacePath, replaceVersion, _ := module.SplitPathVersion(replace)
		tmpMod = append(tmpMod, []byte(fmt.Sprintf("replace %s => %s %s\n", targetPath, replacePath, replaceVersion))...)
	}

	// Write the temporary module file
	tmpModPath := filepath.Join(tmpDir, modFile)
	if err := ioutil.WriteFile(tmpModPath, tmpMod, 0666); err != nil {
		log.Error(err)
		return nil, nil, err
	}

	// Add the target dependency to the temporary module and download the target module
	if _, err := m.exec(tmpDir, "go", "get", "-d", target); err != nil {
		log.Error(err)
		return nil, nil, err
	}

	// Read the updated go.mod for the temporary module
	tmpMod, err = ioutil.ReadFile(tmpModPath)
	if err != nil {
		log.Error(err)
		return nil, nil, err
	}

	// Parse the updated go.mod for the temporary module
	tmpModFile, err := modfile.Parse(tmpModPath, tmpMod, nil)
	if err != nil {
		log.Error(err)
		return nil, nil, err
	}

	// Determine the path/version for the target module
	var modPath string
	var modVersion string
	if replace == "" {
		for _, require := range tmpModFile.Require {
			if require.Mod.Path == targetPath {
				modPath = require.Mod.Path
				modVersion = require.Mod.Version
				break
			}
		}
	} else {
		for _, replace := range tmpModFile.Replace {
			if replace.Old.Path == targetPath {
				modPath = replace.New.Path
				modVersion = replace.New.Version
				break
			}
		}
	}

	// Compute the hash for the target module
	hashBytes := append([]byte(modPath), []byte(modVersion)...)
	modSum := md5.Sum(hashBytes)
	modHash := make(Hash, len(modSum))
	for i, b := range modSum {
		modHash[i] = b
	}

	// Encode the target dependency path
	encPath, err := module.EncodePath(modPath)
	if err != nil {
		log.Error(err)
		return nil, nil, err
	}
	modPath = encPath

	// Lookup the Go cache from the environment
	modCache, err := m.getGoModCacheDir()
	if err != nil {
		log.Error(err)
		return nil, nil, err
	}

	// Read the target go.mod from the cache
	cacheModPath := filepath.Join(modCache, "cache", "download", modPath, "@v", modVersion+".mod")
	modBytes, err := ioutil.ReadFile(cacheModPath)
	if err != nil {
		log.Error(err)
		return nil, nil, err
	}

	// Parse the target go.mod
	modFile, err := modfile.Parse(cacheModPath, modBytes, nil)
	if err != nil {
		log.Error(err)
		return nil, nil, err
	}

	// Format the target go.mod file
	modBytes, err = modFile.Format()
	if err != nil {
		log.Error(err)
		return nil, nil, err
	}
	return modFile, modHash, nil
}

func (m *Manager) getModPath() string {
	return filepath.Join(m.Config.Path, modFile)
}

func (m *Manager) getHashPath() string {
	return filepath.Join(m.Config.Path, hashFile)
}

type goEnv struct {
	GOPATH     string
	GOMODCACHE string
}
