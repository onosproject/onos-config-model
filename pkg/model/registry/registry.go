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

package modelregistry

import (
	"encoding/json"
	"fmt"
	"github.com/onosproject/onos-config-model/pkg/model"
	"github.com/onosproject/onos-config-model/pkg/model/plugin"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/rogpeppe/go-internal/module"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const jsonExt = ".json"

const (
	modelRegistryEnv = "CONFIG_MODEL_REGISTRY"
	targetModuleEnv  = "CONFIG_MODULE_TARGET"
	replaceModuleEnv = "CONFIG_MODULE_REPLACE"
)

var log = logging.GetLogger("config-model", "registry")

// Config is a model plugin registry config
type Config struct {
	Path string `yaml:"path" json:"path"`
}

// NewConfigModelRegistry creates a new config model registry
func NewConfigModelRegistry(config Config) *ConfigModelRegistry {
	if _, err := os.Stat(config.Path); os.IsNotExist(err) {
		err = os.MkdirAll(config.Path, os.ModePerm)
		if err != nil {
			log.Error(err)
		}
	}
	return &ConfigModelRegistry{
		Config: config,
	}
}

// NewConfigModelRegistryFromEnv creates a new config model registry from the environment
func NewConfigModelRegistryFromEnv() *ConfigModelRegistry {
	dir, target, replace := os.Getenv(modelRegistryEnv), os.Getenv(targetModuleEnv), os.Getenv(replaceModuleEnv)
	path, err := GetPath(dir, target, replace)
	if err != nil {
		panic(err)
	}
	return NewConfigModelRegistry(Config{Path: path})
}

// ConfigModelRegistry is a registry of config models
type ConfigModelRegistry struct {
	Config Config
}

// GetModel gets a model by name and version
func (r *ConfigModelRegistry) GetModel(name configmodel.Name, version configmodel.Version) (configmodel.ModelInfo, error) {
	return loadModel(r.getDescriptorFile(name, version))
}

// ListModels lists models in the registry
func (r *ConfigModelRegistry) ListModels() ([]configmodel.ModelInfo, error) {
	return loadModels(r.Config.Path)
}

// AddModel adds a model to the registry
func (r *ConfigModelRegistry) AddModel(model configmodel.ModelInfo) error {
	log.Debugf("Adding model '%s/%s' to registry '%s'", model.Name, model.Version, r.Config.Path)
	bytes, err := json.MarshalIndent(model, "", "  ")
	if err != nil {
		log.Errorf("Adding model '%s/%s' failed: %v", model.Name, model.Version, err)
		return err
	}
	path := r.getDescriptorFile(model.Name, model.Version)
	if err := ioutil.WriteFile(path, bytes, os.ModePerm); err != nil {
		log.Errorf("Adding model '%s/%s' failed: %v", model.Name, model.Version, err)
		return err
	}
	log.Infof("Model '%s/%s' added to registry '%s'", model.Name, model.Version, r.Config.Path)
	return nil
}

// RemoveModel removes a model from the registry
func (r *ConfigModelRegistry) RemoveModel(name configmodel.Name, version configmodel.Version) error {
	log.Debugf("Deleting model '%s/%s' from registry '%s'", name, version, r.Config.Path)
	path := r.getDescriptorFile(name, version)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		if err := os.Remove(path); err != nil {
			log.Errorf("Deleting model '%s/%s' failed: %v", name, version, err)
			return err
		}
	}
	log.Infof("Model '%s/%s' deleted from registry '%s'", name, version, r.Config.Path)
	return nil
}

// LoadPlugin loads a plugin from the registry
func (r *ConfigModelRegistry) LoadPlugin(name configmodel.Name, version configmodel.Version) (modelplugin.ConfigModelPlugin, error) {
	return modelplugin.Load(r.getPluginFile(name, version))
}

func (r *ConfigModelRegistry) getPluginFile(name configmodel.Name, version configmodel.Version) string {
	return filepath.Join(r.Config.Path, fmt.Sprintf("%s-%s.so", name, version))
}

func (r *ConfigModelRegistry) getDescriptorFile(name configmodel.Name, version configmodel.Version) string {
	return filepath.Join(r.Config.Path, fmt.Sprintf("%s-%s.json", name, version))
}

func loadModels(path string) ([]configmodel.ModelInfo, error) {
	var modelFiles []string
	err := filepath.Walk(path, func(file string, info os.FileInfo, err error) error {
		if err == nil && strings.HasSuffix(file, jsonExt) {
			modelFiles = append(modelFiles, file)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	var models []configmodel.ModelInfo
	for _, file := range modelFiles {
		model, err := loadModel(file)
		if err != nil {
			return nil, err
		}
		models = append(models, model)
	}
	return models, nil
}

func loadModel(path string) (configmodel.ModelInfo, error) {
	log.Debugf("Loading model definition '%s'", path)
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		log.Errorf("Failed loading model '%s': %v", path, err)
		return configmodel.ModelInfo{}, err
	}
	modelInfo := configmodel.ModelInfo{}
	err = json.Unmarshal(bytes, &modelInfo)
	if err != nil {
		log.Errorf("Failed decoding model definition '%s': %v", path, err)
		return configmodel.ModelInfo{}, err
	}
	if modelInfo.Name == "" || modelInfo.Version == "" {
		err = errors.NewInvalid("%s is not a valid model descriptor", path)
		log.Errorf("Failed decoding model definition '%s': %v", path, err)
		return configmodel.ModelInfo{}, err
	}
	log.Infof("Loaded model '%s': %s", path, bytes)
	return modelInfo, nil
}

// GetPath :
func GetPath(dir, target, replace string) (string, error) {
	if dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		dir = cwd
	}

	var path string
	var version string
	if replace == "" {
		if i := strings.Index(target, "@"); i >= 0 {
			path = target[:i]
			version = target[i+1:]
		} else {
			path = target
		}
	} else {
		if i := strings.Index(replace, "@"); i >= 0 {
			path = replace[:i]
			version = replace[i+1:]
		} else {
			path = replace
		}
	}

	encPath, err := module.EncodePath(path)
	if err != nil {
		return "", err
	}

	elems := []string{dir, encPath}
	if version != "" {
		elems = append(elems, fmt.Sprintf("@%s", version))
	}
	return filepath.Join(elems...), nil
}
