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

package plugincompiler

import (
	"encoding/json"
	"fmt"
	"github.com/onosproject/onos-config-model/pkg/model"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/rogpeppe/go-internal/modfile"
	"github.com/rogpeppe/go-internal/module"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	_ "github.com/openconfig/gnmi/proto/gnmi" // gnmi
	_ "github.com/openconfig/goyang/pkg/yang" // yang
	_ "github.com/openconfig/ygot/genutil"    // genutil
	_ "github.com/openconfig/ygot/ygen"       // ygen
	_ "github.com/openconfig/ygot/ygot"       // ygot
	_ "github.com/openconfig/ygot/ytypes"     // ytypes
	_ "google.golang.org/protobuf/proto"      // proto
)

var log = logging.GetLogger("config-model", "compiler")

const versionFile = "VERSION"

const (
	modelDir = "model"
	yangDir  = "yang"
)

const (
	modTemplate    = "go.mod.tpl"
	mainTemplate   = "main.go.tpl"
	pluginTemplate = "plugin.go.tpl"
	modelTemplate  = "model.go.tpl"
)

const (
	modFile       = "go.mod"
	mainFile      = "main.go"
	pluginFile    = "plugin.go"
	modelFile     = "model.go"
	modVersionSep = "@"
)

var (
	_, b, _, _            = runtime.Caller(0)
	moduleRoot            = filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(b)))))
	moduleVersionBytes, _ = ioutil.ReadFile(filepath.Join(moduleRoot, versionFile))
	moduleVersion         = strings.TrimSpace(string(moduleVersionBytes))
)

const devSuffix = "-dev"

func getModuleVersion() string {
	return fmt.Sprintf("v%s", moduleVersion)
}

func isReleaseVersion() bool {
	return !strings.HasSuffix(moduleVersion, devSuffix)
}

// CompilerInfo is the compiler info
type CompilerInfo struct {
	Version   string
	IsRelease bool
	Root      string
}

// TemplateInfo provides all the variables for templates
type TemplateInfo struct {
	Model    configmodel.ModelInfo
	Compiler CompilerInfo
}

// CompilePlugin compiles a model plugin to the given path
func CompilePlugin(model configmodel.ModelInfo, config CompilerConfig) error {
	return NewPluginCompiler(config).CompilePlugin(model)
}

// CompilerConfig is a plugin compiler configuration
type CompilerConfig struct {
	TemplatePath string
	BuildPath    string
	OutputPath   string
	Target       string
	Replace      string
}

// NewPluginCompiler creates a new model plugin compiler
func NewPluginCompiler(config CompilerConfig) *PluginCompiler {
	return &PluginCompiler{config}
}

// PluginCompiler is a model plugin compiler
type PluginCompiler struct {
	Config CompilerConfig
}

// CompilePlugin compiles a model plugin to the given path
func (c *PluginCompiler) CompilePlugin(model configmodel.ModelInfo) error {
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
	if err := c.generateModelPlugin(model); err != nil {
		log.Errorf("Compiling ConfigModel '%s/%s' failed: %v", model.Name, model.Version, err)
		return err
	}

	// Generate the YANG bindings
	c.createDir(c.getYangDir(model))
	if err := c.copyFiles(model); err != nil {
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

func (c *PluginCompiler) getTemplateInfo(model configmodel.ModelInfo) (TemplateInfo, error) {
	return TemplateInfo{
		Model: model,
		Compiler: CompilerInfo{
			Version:   getModuleVersion(),
			IsRelease: isReleaseVersion(),
			Root:      moduleRoot,
		},
	}, nil
}

func (c *PluginCompiler) getPluginPath(model configmodel.ModelInfo) string {
	return filepath.Join(c.Config.OutputPath, fmt.Sprintf("%s-%s.so", model.Plugin.Name, model.Plugin.Version))
}

func (c *PluginCompiler) getPluginMod(model configmodel.ModelInfo) string {
	return fmt.Sprintf("github.com/onosproject/onos-config-model/%s", c.getSafeQualifiedName(model))
}

func (c *PluginCompiler) compilePlugin(model configmodel.ModelInfo) error {
	path := c.getPluginPath(model)
	log.Infof("Compiling plugin '%s'", path)
	_, err := c.exec(c.getModuleDir(model), "go", "build", "-o", path, "-buildmode=plugin", c.getPluginMod(model))
	if err != nil {
		log.Errorf("Compiling plugin '%s' failed: %v", path, err)
		return err
	}
	return nil
}

func (c *PluginCompiler) exec(dir string, name string, args ...string) (string, error) {
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

func (c *PluginCompiler) cleanBuild(model configmodel.ModelInfo) error {
	if _, err := os.Stat(c.getModuleDir(model)); err == nil {
		return os.RemoveAll(c.getModuleDir(model))
	}
	return nil
}

func (c *PluginCompiler) copyFiles(model configmodel.ModelInfo) error {
	for _, file := range model.Files {
		if err := c.copyFile(model, file); err != nil {
			return err
		}
	}
	return nil
}

func (c *PluginCompiler) copyFile(model configmodel.ModelInfo, file configmodel.FileInfo) error {
	path := c.getYangPath(model, file)
	log.Debugf("Copying YANG module '%s' to '%s'", file.Path, path)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := ioutil.WriteFile(path, file.Data, os.ModePerm)
		if err != nil {
			log.Errorf("Copying YANG module '%s' failed: %v", file.Path, err)
			return err
		}
	}
	return nil
}

func (c *PluginCompiler) generateYangBindings(model configmodel.ModelInfo) error {
	path := filepath.Join(c.getModelPath(model, "generated.go"))
	log.Debugf("Generating YANG bindings '%s'", path)
	args := []string{
		"run",
		"github.com/openconfig/ygot/generator",
		"-path=yang",
		"-output_file=model/generated.go",
		"-package_name=configmodel",
		"-generate_fakeroot",
	}

	for _, module := range model.Modules {
		args = append(args, module.File)
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

func (c *PluginCompiler) generateMain(model configmodel.ModelInfo) error {
	info, err := c.getTemplateInfo(model)
	if err != nil {
		return err
	}
	return applyTemplate(mainTemplate, c.getTemplatePath(mainTemplate), c.getModulePath(model, mainFile), info)
}

func (c *PluginCompiler) generateTemplate(model configmodel.ModelInfo, template, inPath, outPath string) error {
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

func (c *PluginCompiler) generateMod(model configmodel.ModelInfo) error {
	if c.Config.Target == "" {
		return c.generateTemplate(model, modTemplate, c.getTemplatePath(modTemplate), c.getModulePath(model, modFile))
	}
	return c.fetchMod(model)
}

func (c *PluginCompiler) getGoEnv() (goEnv, error) {
	wd, err := os.Getwd()
	if err != nil {
		return goEnv{}, err
	}

	envJSON, err := c.exec(wd, "go", "env", "-json", "GOPATH", "GOMODCACHE")
	if err != nil {
		return goEnv{}, err
	}
	env := goEnv{}
	if err := json.Unmarshal([]byte(envJSON), &env); err != nil {
		return goEnv{}, err
	}
	return env, nil
}

func (c *PluginCompiler) getGoModCacheDir() (string, error) {
	env, err := c.getGoEnv()
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

func (c *PluginCompiler) fetchMod(model configmodel.ModelInfo) error {
	log.Debugf("Generating '%s'", c.getModulePath(model, modFile))
	tmpDir, err := ioutil.TempDir("", c.getSafeQualifiedName(model))
	if err != nil {
		log.Error(err)
		return err
	}
	defer os.RemoveAll(tmpDir)

	target := c.Config.Target
	replace := c.Config.Replace
	targetPath, _ := splitModPathVersion(target)

	// Generate a temporary module with which to pull the target module
	tmpMod := []byte("module m\n")
	if replace != "" {
		replacePath, replaceVersion := splitModPathVersion(replace)
		tmpMod = append(tmpMod, []byte(fmt.Sprintf("replace %s => %s %s\n", targetPath, replacePath, replaceVersion))...)
	}

	// Write the temporary module file
	tmpModPath := filepath.Join(tmpDir, modFile)
	if err := ioutil.WriteFile(tmpModPath, tmpMod, 0666); err != nil {
		log.Error(err)
		return err
	}

	// Add the target dependency to the temporary module and download the target module
	if _, err := c.exec(tmpDir, "go", "get", "-d", target); err != nil {
		log.Error(err)
		return err
	}

	// Read the updated go.mod for the temporary module
	tmpMod, err = ioutil.ReadFile(tmpModPath)
	if err != nil {
		log.Error(err)
		return err
	}

	// Parse the updated go.mod for the temporary module
	tmpModFile, err := modfile.Parse(tmpModPath, tmpMod, nil)
	if err != nil {
		log.Error(err)
		return err
	}

	// Determine the path/version for the target dependency
	var depPath string
	var depVersion string
	if replace == "" {
		for _, require := range tmpModFile.Require {
			if require.Mod.Path == targetPath {
				depPath = require.Mod.Path
				depVersion = require.Mod.Version
				break
			}
		}
	} else {
		for _, replace := range tmpModFile.Replace {
			if replace.Old.Path == targetPath {
				depPath = replace.New.Path
				depVersion = replace.New.Version
				break
			}
		}
	}

	// Encode the target dependency path
	encPath, err := module.EncodePath(depPath)
	if err != nil {
		log.Error(err)
		return err
	}
	depPath = encPath

	// Lookup the Go cache from the environment
	modCache, err := c.getGoModCacheDir()
	if err != nil {
		log.Error(err)
		return err
	}

	// Read the target dependency go.mod from the cache
	depModPath := filepath.Join(modCache, "cache", "download", depPath, "@v", depVersion+".mod")
	depMod, err := ioutil.ReadFile(depModPath)
	if err != nil {
		log.Error(err)
		return err
	}

	// Parse the target dependency go.mod
	depModFile, err := modfile.Parse(depModPath, depMod, nil)
	if err != nil {
		log.Error(err)
		return err
	}

	// Rename the target dependency module to adopt its dependencies for the plugin module
	pluginModFile := depModFile
	if err := pluginModFile.AddModuleStmt(c.getPluginMod(model)); err != nil {
		return err
	}

	// Format the updated plugin go.mod
	pluginMod, err := pluginModFile.Format()
	if err != nil {
		log.Error(err)
		return err
	}

	// Write the plugin module go.mod
	pluginModPath := c.getModulePath(model, modFile)
	if err := ioutil.WriteFile(pluginModPath, pluginMod, 0666); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (c *PluginCompiler) generateModelPlugin(model configmodel.ModelInfo) error {
	return c.generateTemplate(model, pluginTemplate, c.getTemplatePath(pluginTemplate), c.getModelPath(model, pluginFile))
}

func (c *PluginCompiler) generateConfigModel(model configmodel.ModelInfo) error {
	return c.generateTemplate(model, modelTemplate, c.getTemplatePath(modelTemplate), c.getModelPath(model, modelFile))
}

func (c *PluginCompiler) getModuleDir(model configmodel.ModelInfo) string {
	return filepath.Join(c.Config.BuildPath, c.getSafeQualifiedName(model))
}

func (c *PluginCompiler) getModulePath(model configmodel.ModelInfo, name string) string {
	return filepath.Join(c.getModuleDir(model), name)
}

func (c *PluginCompiler) getModelDir(model configmodel.ModelInfo) string {
	return filepath.Join(c.getModuleDir(model), modelDir)
}

func (c *PluginCompiler) getModelPath(model configmodel.ModelInfo, name string) string {
	return filepath.Join(c.getModelDir(model), name)
}

func (c *PluginCompiler) getYangDir(model configmodel.ModelInfo) string {
	return filepath.Join(c.getModuleDir(model), yangDir)
}

func (c *PluginCompiler) getYangPath(model configmodel.ModelInfo, file configmodel.FileInfo) string {
	return filepath.Join(c.getYangDir(model), filepath.Base(file.Path))
}

func (c *PluginCompiler) getSafeQualifiedName(model configmodel.ModelInfo) string {
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
