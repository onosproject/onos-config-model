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

package repository

import (
	"encoding/json"
	"fmt"
	"github.com/onosproject/onos-config-model-go/pkg/model"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const jsonExt = ".json"

var log = logging.GetLogger("config-model", "repository")

// Config is a model plugin repository config
type Config struct {
	Path string `yaml:"path" json:"path"`
}

// NewRepository creates a new config model repository
func NewRepository(config Config) *Repository {
	return &Repository{
		Config: config,
	}
}

// Repository is a repository of config models
type Repository struct {
	Config Config
}

// GetModel gets a model by name and version
func (r *Repository) GetModel(name model.Name, version model.Version) (model.ConfigModelInfo, error) {
	return loadModel(r.getDescriptorFile(name, version))
}

// ListModels lists models in the repository
func (r *Repository) ListModels() ([]model.ConfigModelInfo, error) {
	return loadModels(r.Config.Path)
}

// addModel adds a model to the repository
func (r *Repository) addModel(model model.ConfigModelInfo) error {
	log.Debugf("Adding model '%s/%s' to repository '%s'", model.Name, model.Version, r.Config.Path)
	bytes, err := json.MarshalIndent(model, "", "  ")
	if err != nil {
		log.Errorf("Adding model '%s/%s' failed: %v", model.Name, model.Version, err)
		return err
	}
	path := r.getDescriptorFile(model.Name, model.Version)
	if err := ioutil.WriteFile(path, bytes, 0666); err != nil {
		log.Errorf("Adding model '%s/%s' failed: %v", model.Name, model.Version, err)
		return err
	}
	log.Infof("Model '%s/%s' added to repository '%s'", model.Name, model.Version, r.Config.Path)
	return nil
}

// removeModel removes a model from the registry
func (r *Repository) removeModel(name model.Name, version model.Version) error {
	log.Debugf("Deleting model '%s/%s' from repository '%s'", name, version, r.Config.Path)
	path := r.getDescriptorFile(name, version)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		if err := os.Remove(path); err != nil {
			log.Errorf("Deleting model '%s/%s' failed: %v", name, version, err)
			return err
		}
	}
	log.Infof("Model '%s/%s' deleted from repository '%s'", name, version, r.Config.Path)
	return nil
}

func (r *Repository) getDescriptorFile(name model.Name, version model.Version) string {
	return filepath.Join(r.Config.Path, fmt.Sprintf("%s-%s.json", name, version))
}

func loadModels(path string) ([]model.ConfigModelInfo, error) {
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

	var models []model.ConfigModelInfo
	for _, file := range modelFiles {
		model, err := loadModel(file)
		if err != nil {
			return nil, err
		}
		models = append(models, model)
	}
	return models, nil
}

func loadModel(path string) (model.ConfigModelInfo, error) {
	log.Debugf("Loading model definition '%s'", path)
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		log.Errorf("Failed loading model '%s': %v", path, err)
		return model.ConfigModelInfo{}, err
	}
	modelInfo := model.ConfigModelInfo{}
	err = json.Unmarshal(bytes, &modelInfo)
	if err != nil {
		log.Errorf("Failed decoding model definition '%s': %v", path, err)
		return model.ConfigModelInfo{}, err
	}
	if modelInfo.Name == "" || modelInfo.Version == "" {
		err = errors.NewInvalid("%s is not a valid model descriptor", path)
		log.Errorf("Failed decoding model definition '%s': %v", path, err)
		return model.ConfigModelInfo{}, err
	}
	log.Infof("Loaded model '%s': %s", path, bytes)
	return modelInfo, nil
}
