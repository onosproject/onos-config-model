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

package compiler

import (
	"fmt"
	"github.com/onosproject/onos-config-model-go/pkg/model"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	_ "github.com/golang/protobuf/proto"
	_ "github.com/openconfig/gnmi/proto/gnmi"
	_ "github.com/openconfig/goyang/pkg/yang"
	_ "github.com/openconfig/ygot/genutil"
	_ "github.com/openconfig/ygot/ygen"
	_ "github.com/openconfig/ygot/ygot"
	_ "github.com/openconfig/ygot/ytypes"
)

var log = logging.GetLogger("config-model", "compiler")

const (
	modelDir        = "model"
	yangDir         = "yang"
	compilerVersion = "v0.6.10"
)

const (
	modTemplate          = "go.mod.tpl"
	mainTemplate         = "main.go.tpl"
	pluginTemplate       = "plugin.go.tpl"
	modelTemplate        = "model.go.tpl"
	unmarshallerTemplate = "unmarshaller.go.tpl"
	validatorTemplate    = "validator.go.tpl"
)

const (
	modFile          = "go.mod"
	mainFile         = "main.go"
	pluginFile       = "plugin.go"
	modelFile        = "model.go"
	unmarshallerFile = "unmarshaller.go"
	validatorFile    = "validator.go"
)

// CompilerInfo is the compiler info
type CompilerInfo struct {
	Version string
	Root    string
}

// TemplateInfo provides all the variables for templates
type TemplateInfo struct {
	Model    model.ConfigModelInfo
	Compiler CompilerInfo
}

// CompilePlugin compiles a model plugin to the given path
func CompilePlugin(model model.ConfigModelInfo, config PluginCompilerConfig) error {
	return NewPluginCompiler(config).CompilePlugin(model)
}

// PluginCompilerConfig is a plugin compiler configuration
type PluginCompilerConfig struct {
	TemplatePath string
	BuildPath    string
	OutputPath   string
}

// NewPluginCompiler creates a new model plugin compiler
func NewPluginCompiler(config PluginCompilerConfig) *PluginCompiler {
	return &PluginCompiler{config}
}

// PluginCompiler is a model plugin compiler
type PluginCompiler struct {
	Config PluginCompilerConfig
}

// CompilePlugin compiles a model plugin to the given path
func (c *PluginCompiler) CompilePlugin(model model.ConfigModelInfo) error {
	log.Infof("Compiling ConfigModel '%s/%s' to '%s'", model.Name, model.Version, c.getPluginPath(model))

	// Ensure the build directory exists
	c.createDir(c.Config.BuildPath)

	// Create the module files
	c.createDir(c.getModuleDir(model))
	if err := c.generateMod(model); err != nil {
		log.Errorf("Compiling ConfigModel '%s/%s' failed: %v", model.Name, model.Version, err)
		return err
	}
	if err := c.generateMain(model); err != nil {
		log.Errorf("Compiling ConfigModel '%s/%s' failed: %v", model.Name, model.Version, err)
		return err
	}

	// Create the model plugin
	c.createDir(c.getModelDir(model))
	if err := c.generateConfigModel(model); err != nil {
		log.Errorf("Compiling ConfigModel '%s/%s' failed: %v", model.Name, model.Version, err)
		return err
	}
	if err := c.generateUnmarshaller(model); err != nil {
		log.Errorf("Compiling ConfigModel '%s/%s' failed: %v", model.Name, model.Version, err)
		return err
	}
	if err := c.generateValidator(model); err != nil {
		log.Errorf("Compiling ConfigModel '%s/%s' failed: %v", model.Name, model.Version, err)
		return err
	}
	if err := c.generateModelPlugin(model); err != nil {
		log.Errorf("Compiling ConfigModel '%s/%s' failed: %v", model.Name, model.Version, err)
		return err
	}

	// Generate the YANG bindings
	c.createDir(c.getYangDir(model))
	if err := c.copyModules(model); err != nil {
		log.Errorf("Compiling ConfigModel '%s/%s' failed: %v", model.Name, model.Version, err)
		return err
	}
	if err := c.generateYangBindings(model); err != nil {
		log.Errorf("Compiling ConfigModel '%s/%s' failed: %v", model.Name, model.Version, err)
		return err
	}

	// Compile the plugin
	c.createDir(c.Config.OutputPath)
	if err := c.compilePlugin(model); err != nil {
		log.Errorf("Compiling ConfigModel '%s/%s' failed: %v", model.Name, model.Version, err)
		return err
	}

	// Clean up the build
	if err := c.cleanBuild(model); err != nil {
		log.Errorf("Compiling ConfigModel '%s/%s' failed: %v", model.Name, model.Version, err)
		return err
	}
	return nil
}

func (c *PluginCompiler) getTemplateInfo(model model.ConfigModelInfo) (TemplateInfo, error) {
	wd, err := os.Getwd()
	if err != nil {
		return TemplateInfo{}, err
	}
	return TemplateInfo{
		Model: model,
		Compiler: CompilerInfo{
			Version: compilerVersion,
			Root:    wd,
		},
	}, nil
}

func (c *PluginCompiler) getPluginPath(model model.ConfigModelInfo) string {
	return filepath.Join(c.Config.OutputPath, model.Plugin.File)
}

func (c *PluginCompiler) compilePlugin(model model.ConfigModelInfo) error {
	log.Infof("Compiling plugin '%s'", c.getPluginPath(model))
	cmd := exec.Command("go", "build", "-o", c.getPluginPath(model), "-buildmode=plugin", "github.com/onosproject/onos-config-model-go/"+c.getSafeQualifiedName(model))
	cmd.Dir = c.getModuleDir(model)
	cmd.Env = append(os.Environ(), "GO111MODULE=on", "CGO_ENABLED=1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("Compiling plugin '%s' failed: %v", c.getPluginPath(model), err)
		return err
	}
	return nil
}

func (c *PluginCompiler) cleanBuild(model model.ConfigModelInfo) error {
	if _, err := os.Stat(c.getModuleDir(model)); err == nil {
		return os.RemoveAll(c.getModuleDir(model))
	}
	return nil
}

func (c *PluginCompiler) copyModules(model model.ConfigModelInfo) error {
	for _, module := range model.Modules {
		if err := c.copyModule(model, module); err != nil {
			return err
		}
	}
	return nil
}

func (c *PluginCompiler) copyModule(model model.ConfigModelInfo, module model.ConfigModuleInfo) error {
	path := c.getYangPath(model, module)
	log.Debugf("Copying YANG module '%s/%s' to '%s'", module.Name, module.Version, path)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := ioutil.WriteFile(path, []byte(module.Data), os.ModePerm)
		if err != nil {
			log.Errorf("Copying YANG module '%s/%s' failed: %v", module.Name, module.Version, err)
			return err
		}
	}
	return nil
}

func (c *PluginCompiler) generateYangBindings(model model.ConfigModelInfo) error {
	path := filepath.Join(c.getModelPath(model, "generated.go"))
	log.Debugf("Generating YANG bindings '%s'", path)
	args := []string{
		"run",
		"github.com/openconfig/ygot/generator",
		"-path=yang",
		"-output_file=model/generated.go",
		"-package_name=model",
		"-generate_fakeroot",
	}

	for _, module := range model.Modules {
		args = append(args, c.getYangFile(module))
	}

	cmd := exec.Command("go", args...)
	cmd.Dir = c.getModuleDir(model)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("Generating YANG bindings '%s' failed: %v", path, err)
		return err
	}
	return nil
}

func (c *PluginCompiler) getTemplatePath(name string) string {
	return filepath.Join(c.Config.TemplatePath, name)
}

func (c *PluginCompiler) generateMain(model model.ConfigModelInfo) error {
	info, err := c.getTemplateInfo(model)
	if err != nil {
		return err
	}
	return applyTemplate(mainTemplate, c.getTemplatePath(mainTemplate), c.getModulePath(model, mainFile), info)
}

func (c *PluginCompiler) generateTemplate(model model.ConfigModelInfo, template, inPath, outPath string) error {
	log.Debugf("Generating '%s'", outPath)
	info, err := c.getTemplateInfo(model)
	if err != nil {
		log.Errorf("Generating '%s' failed: %v", outPath, err)
		return err
	}
	if err := applyTemplate(template, inPath, outPath, info); err != nil {
		log.Errorf("Generating '%s' failed: %v", outPath, err)
		return err
	}
	return nil
}

func (c *PluginCompiler) generateMod(model model.ConfigModelInfo) error {
	return c.generateTemplate(model, modTemplate, c.getTemplatePath(modTemplate), c.getModulePath(model, modFile))
}

func (c *PluginCompiler) generateModelPlugin(model model.ConfigModelInfo) error {
	return c.generateTemplate(model, pluginTemplate, c.getTemplatePath(pluginTemplate), c.getModelPath(model, pluginFile))
}

func (c *PluginCompiler) generateConfigModel(model model.ConfigModelInfo) error {
	return c.generateTemplate(model, modelTemplate, c.getTemplatePath(modelTemplate), c.getModelPath(model, modelFile))
}

func (c *PluginCompiler) generateUnmarshaller(model model.ConfigModelInfo) error {
	return c.generateTemplate(model, unmarshallerTemplate, c.getTemplatePath(unmarshallerTemplate), c.getModelPath(model, unmarshallerFile))
}

func (c *PluginCompiler) generateValidator(model model.ConfigModelInfo) error {
	return c.generateTemplate(model, validatorTemplate, c.getTemplatePath(validatorTemplate), c.getModelPath(model, validatorFile))
}

func (c *PluginCompiler) getModuleDir(model model.ConfigModelInfo) string {
	return filepath.Join(c.Config.BuildPath, c.getSafeQualifiedName(model))
}

func (c *PluginCompiler) getModulePath(model model.ConfigModelInfo, name string) string {
	return filepath.Join(c.getModuleDir(model), name)
}

func (c *PluginCompiler) getModelDir(model model.ConfigModelInfo) string {
	return filepath.Join(c.getModuleDir(model), modelDir)
}

func (c *PluginCompiler) getModelPath(model model.ConfigModelInfo, name string) string {
	return filepath.Join(c.getModelDir(model), name)
}

func (c *PluginCompiler) getYangDir(model model.ConfigModelInfo) string {
	return filepath.Join(c.getModuleDir(model), yangDir)
}

func (c *PluginCompiler) getYangPath(model model.ConfigModelInfo, module model.ConfigModuleInfo) string {
	return filepath.Join(c.getYangDir(model), c.getYangFile(module))
}

func (c *PluginCompiler) getYangFile(module model.ConfigModuleInfo) string {
	return fmt.Sprintf("%s@%s.yang", module.Name, module.Version)
}

func (c *PluginCompiler) getSafeQualifiedName(model model.ConfigModelInfo) string {
	return strings.ReplaceAll(fmt.Sprintf("%s_%s", model.Name, model.Version), ".", "_")
}

func (c *PluginCompiler) createDir(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		log.Debugf("Creating '%s'", dir)
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			log.Errorf("Creating '%s' failed: %v", dir, err)
		}
	}
}

func (c *PluginCompiler) removeDir(dir string) {
	if _, err := os.Stat(dir); err == nil {
		log.Debugf("Removing '%s'", dir)
		if err := os.RemoveAll(dir); err != nil {
			log.Errorf("Removing '%s' failed: %v", dir, err)
		}
	}
}

func applyTemplate(name, tplPath, outPath string, data TemplateInfo) error {
	var funcs template.FuncMap = map[string]interface{}{
		"quote": func(value interface{}) string {
			return fmt.Sprintf("\"%s\"", value)
		},
		"replace": func(search, replace string, value interface{}) string {
			return strings.ReplaceAll(fmt.Sprint(value), search, replace)
		},
	}

	tpl, err := template.New(name).
		Funcs(funcs).
		ParseFiles(tplPath)
	if err != nil {
		return err
	}

	file, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return tpl.Execute(file, data)
}
