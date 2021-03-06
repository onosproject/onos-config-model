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

package pluginmodule

import (
	"encoding/json"
	"fmt"
	"github.com/onosproject/onos-lib-go/pkg/errors"
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
	"strings"
)

var log = logging.GetLogger("config-model", "plugin", "module")

const (
	defaultPath   = "/etc/onos/mod"
	modFile       = "go.mod"
	hashFile      = "mod.md5"
	modVersionSep = "@"
)

// Hash is a module hash
type Hash []byte

// ResolverConfig is a module resolver configuration
type ResolverConfig struct {
	Path    string
	Target  string
	Replace string
}

// NewResolver creates a new module resolver
func NewResolver(config ResolverConfig) *Resolver {
	if config.Path == "" {
		config.Path = defaultPath
	}
	return &Resolver{config}
}

// Resolver is a module resolver
type Resolver struct {
	Config ResolverConfig
}

func (r *Resolver) exec(dir string, name string, args ...string) (string, error) {
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

func (r *Resolver) getGoEnv() (goEnv, error) {
	wd, err := os.Getwd()
	if err != nil {
		return goEnv{}, err
	}

	envJSON, err := r.exec(wd, "go", "env", "-json", "GOPATH", "GOMODCACHE")
	if err != nil {
		return goEnv{}, err
	}
	env := goEnv{}
	if err := json.Unmarshal([]byte(envJSON), &env); err != nil {
		return goEnv{}, err
	}
	return env, nil
}

func (r *Resolver) getGoModCacheDir() (string, error) {
	env, err := r.getGoEnv()
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

func (r *Resolver) Resolve() (*modfile.File, Hash, error) {
	modPath := r.getModPath()
	modBytes, modErr := ioutil.ReadFile(modPath)
	hashPath := r.getHashPath()
	hashBytes, hashErr := ioutil.ReadFile(hashPath)
	if modErr != nil || hashErr != nil {
		mod, hash, err := r.fetchMod()
		if err != nil {
			return nil, nil, err
		}
		modBytes, err := mod.Format()
		if err != nil {
			log.Errorf("Failed to format go.mod: %s", err)
			return nil, nil, err
		}
		if err := ioutil.WriteFile(r.getModPath(), modBytes, 0666); err != nil {
			log.Errorf("Failed to write go.mod: %s", err)
			return nil, nil, err
		}
		if err := ioutil.WriteFile(r.getHashPath(), hash, 0666); err != nil {
			log.Errorf("Failed to write module hash: %s", err)
			return nil, nil, err
		}
		return mod, hash, nil
	}
	modFile, err := modfile.Parse(modPath, modBytes, nil)
	if err != nil {
		log.Errorf("Failed to parse go.mod: %s", err)
		return nil, nil, err
	}
	return modFile, hashBytes, nil
}

func (r *Resolver) fetchMod() (*modfile.File, Hash, error) {
	target, replace := r.Config.Target, r.Config.Replace
	if target == "" {
		err := errors.NewInvalid("no target module configured")
		log.Errorf("Failed to fetch module '%s': %s", r.Config.Target, err)
		return nil, nil, err
	}

	targetPath, _ := splitModPathVersion(target)

	log.Infof("Fetching module '%s'", target)
	tmpDir, err := ioutil.TempDir("", "config-model")
	if err != nil {
		log.Errorf("Failed to fetch module '%s': %s", r.Config.Target, err)
		return nil, nil, err
	}
	defer os.RemoveAll(tmpDir)

	// Generate a temporary module with which to pull the target module
	tmpMod := []byte("module m\n")
	if replace != "" {
		replacePath, replaceVersion := splitModPathVersion(replace)
		tmpMod = append(tmpMod, []byte(fmt.Sprintf("replace %s => %s %s\n", targetPath, replacePath, replaceVersion))...)
	}

	// Write the temporary module file
	tmpModPath := filepath.Join(tmpDir, modFile)
	if err := ioutil.WriteFile(tmpModPath, tmpMod, 0666); err != nil {
		log.Errorf("Failed to fetch module '%s': %s", r.Config.Target, err)
		return nil, nil, err
	}

	// Add the target dependency to the temporary module and download the target module
	if _, err := r.exec(tmpDir, "go", "get", "-d", target); err != nil {
		log.Errorf("Failed to fetch module '%s': %s", r.Config.Target, err)
		return nil, nil, err
	}

	// Read the updated go.mod for the temporary module
	tmpMod, err = ioutil.ReadFile(tmpModPath)
	if err != nil {
		log.Errorf("Failed to fetch module '%s': %s", r.Config.Target, err)
		return nil, nil, err
	}

	// Parse the updated go.mod for the temporary module
	tmpModFile, err := modfile.Parse(tmpModPath, tmpMod, nil)
	if err != nil {
		log.Errorf("Failed to fetch module '%s': %s", r.Config.Target, err)
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

	// Encode the target dependency path
	encPath, err := module.EncodePath(modPath)
	if err != nil {
		log.Errorf("Failed to fetch module '%s': %s", r.Config.Target, err)
		return nil, nil, err
	}
	modPath = encPath

	// Lookup the Go cache from the environment
	modCache, err := r.getGoModCacheDir()
	if err != nil {
		log.Errorf("Failed to fetch module '%s': %s", r.Config.Target, err)
		return nil, nil, err
	}

	// Read the target go.mod from the cache
	cacheModPath := filepath.Join(modCache, "cache", "download", modPath, "@v", modVersion+".mod")
	modBytes, err := ioutil.ReadFile(cacheModPath)
	if err != nil {
		log.Errorf("Failed to fetch module '%s': %s", r.Config.Target, err)
		return nil, nil, err
	}

	// Parse the target go.mod
	modFile, err := modfile.Parse(cacheModPath, modBytes, nil)
	if err != nil {
		log.Errorf("Failed to fetch module '%s': %s", r.Config.Target, err)
		return nil, nil, err
	}

	// Format the target go.mod file
	modBytes, err = modFile.Format()
	if err != nil {
		log.Errorf("Failed to fetch module '%s': %s", r.Config.Target, err)
		return nil, nil, err
	}

	// Read the target ziphash from the cache
	hashPath := filepath.Join(modCache, "cache", "download", modPath, "@v", modVersion+".ziphash")
	hashBytes, err := ioutil.ReadFile(hashPath)
	if err != nil {
		log.Errorf("Failed to fetch module '%s' hash: %s", r.Config.Target, err)
		return nil, nil, err
	}
	return modFile, Hash(hashBytes), nil
}

func (r *Resolver) getModPath() string {
	return filepath.Join(r.Config.Path, modFile)
}

func (r *Resolver) getHashPath() string {
	return filepath.Join(r.Config.Path, hashFile)
}

func splitModPathVersion(mod string) (string, string) {
	if i := strings.Index(mod, modVersionSep); i >= 0 {
		return mod[:i], mod[i+1:]
	}
	return mod, ""
}

type goEnv struct {
	GOPATH     string
	GOMODCACHE string
}
