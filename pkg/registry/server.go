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

package registry

import (
	"context"
	"fmt"
	"github.com/onosproject/onos-config-model-go/api/onos/configmodel"
	"github.com/onosproject/onos-config-model-go/pkg/compiler"
	"github.com/onosproject/onos-config-model-go/pkg/model"
	"github.com/onosproject/onos-lib-go/pkg/northbound"
	"google.golang.org/grpc"
	"os"
	"path/filepath"
)

func NewService(registry *ConfigModelRegistry, compiler *compiler.PluginCompiler) northbound.Service {
	return &Service{
		registry:     registry,
		compiler: compiler,
	}
}

type Service struct {
	registry     *ConfigModelRegistry
	compiler *compiler.PluginCompiler
}

func (s *Service) Register(r *grpc.Server) {
	server := &Server{
		registry:     s.registry,
		compiler: s.compiler,
	}
	configmodel.RegisterConfigModelRegistryServiceServer(r, server)
}

var _ northbound.Service = &Service{}

// Server is a registry server
type Server struct {
	registry     *ConfigModelRegistry
	compiler *compiler.PluginCompiler
}

func (s *Server) GetModel(ctx context.Context, request *configmodel.GetModelRequest) (*configmodel.GetModelResponse, error) {
	log.Debugf("Received GetModelRequest %+v", request)
	modelInfo, err := s.registry.GetModel(model.Name(request.Name), model.Version(request.Version))
	if err != nil {
		log.Warnf("GetModelRequest %+v failed: %v", request, err)
		return nil, err
	}

	var modules []*configmodel.ConfigModule
	for _, moduleInfo := range modelInfo.Modules {
		modules = append(modules, &configmodel.ConfigModule{
			Name:         string(moduleInfo.Name),
			Organization: string(moduleInfo.Organization),
			Version:      string(moduleInfo.Version),
			Data:         moduleInfo.Data,
		})
	}
	response := &configmodel.GetModelResponse{
		Model: &configmodel.ConfigModel{
			Name:    string(modelInfo.Name),
			Version: string(modelInfo.Version),
			Modules: modules,
		},
	}
	log.Debugf("Sending GetModelResponse %+v", response)
	return response, nil
}

func (s *Server) ListModels(ctx context.Context, request *configmodel.ListModelsRequest) (*configmodel.ListModelsResponse, error) {
	log.Debugf("Received ListModelsRequest %+v", request)
	modelInfos, err := s.registry.ListModels()
	if err != nil {
		log.Warnf("ListModelsRequest %+v failed: %v", request, err)
		return nil, err
	}

	var models []*configmodel.ConfigModel
	for _, modelInfo := range modelInfos {
		var modules []*configmodel.ConfigModule
		for _, module := range modelInfo.Modules {
			modules = append(modules, &configmodel.ConfigModule{
				Name:         string(module.Name),
				Organization: string(module.Organization),
				Version:      string(module.Version),
				Data:         module.Data,
			})
		}
		models = append(models, &configmodel.ConfigModel{
			Name:    string(modelInfo.Name),
			Version: string(modelInfo.Version),
			Modules: modules,
		})
	}
	response := &configmodel.ListModelsResponse{
		Models: models,
	}
	log.Debugf("Sending ListModelsResponse %+v", response)
	return response, nil
}

func (s *Server) PushModel(ctx context.Context, request *configmodel.PushModelRequest) (*configmodel.PushModelResponse, error) {
	log.Debugf("Received PushModelRequest %+v", request)
	var moduleInfos []model.ConfigModuleInfo
	for _, module := range request.Model.Modules {
		moduleInfos = append(moduleInfos, model.ConfigModuleInfo{
			Name:         model.Name(module.Name),
			Organization: module.Organization,
			Version:      model.Version(module.Version),
			Data:         module.Data,
		})
	}
	modelInfo := model.ConfigModelInfo{
		Name:    model.Name(request.Model.Name),
		Version: model.Version(request.Model.Version),
		Modules: moduleInfos,
		Plugin: model.ConfigPluginInfo{
			Name:    model.Name(request.Model.Name),
			Version: model.Version(request.Model.Version),
			File:    getPluginFile(request.Model.Name, request.Model.Version),
		},
	}
	err := s.compiler.CompilePlugin(modelInfo)
	if err != nil {
		log.Warnf("PushModelRequest %+v failed: %v", request, err)
		return nil, err
	}
	err = s.registry.AddModel(modelInfo)
	response := &configmodel.PushModelResponse{}
	log.Debugf("Sending PushModelResponse %+v", response)
	return response, nil
}

func (s *Server) DeleteModel(ctx context.Context, request *configmodel.DeleteModelRequest) (*configmodel.DeleteModelResponse, error) {
	log.Debugf("Received DeleteModelRequest %+v", request)
	err := s.registry.RemoveModel(model.Name(request.Name), model.Version(request.Version))
	if err != nil {
		log.Warnf("DeleteModelRequest %+v failed: %v", request, err)
		return nil, err
	}
	path := filepath.Join(s.registry.Config.Path, getPluginFile(request.Name, request.Version))
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		os.Remove(path)
	}
	response := &configmodel.DeleteModelResponse{}
	log.Debugf("Sending DeleteModelResponse %+v", response)
	return response, nil
}

func getPluginFile(name, version string) string {
	return fmt.Sprintf("%s-%s.so", name, version)
}

var _ configmodel.ConfigModelRegistryServiceServer = &Server{}
