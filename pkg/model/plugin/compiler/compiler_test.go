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
	"github.com/onosproject/onos-config-model/pkg/model"
	"github.com/onosproject/onos-config-model/pkg/model/plugin"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func TestCompiler(t *testing.T) {
	if isReleaseVersion() {
		t.Skip()
	}

	config := CompilerConfig{
		TemplatePath: filepath.Join(moduleRoot, "pkg", "model", "plugin", "compiler", "templates"),
		BuildPath:    filepath.Join(moduleRoot, "build", "_output"),
		OutputPath:   filepath.Join(moduleRoot, "test", "_output"),
	}
	compiler := NewPluginCompiler(config)

	bytes, err := ioutil.ReadFile(filepath.Join(moduleRoot, "test", "test@2020-11-18.yang"))
	assert.NoError(t, err)

	modelInfo := configmodel.ModelInfo{
		Name:    "test",
		Version: "1.0.0",
		Modules: []configmodel.ModuleInfo{
			{
				Name:         "test",
				Organization: "ONF",
				Version:      "2020-11-18",
				Data:         bytes,
			},
		},
		Plugin: configmodel.PluginInfo{
			Name:    "test",
			Version: "1.0.0",
		},
	}
	err = compiler.CompilePlugin(modelInfo)
	assert.NoError(t, err)

	plugin, err := modelplugin.Load(filepath.Join(moduleRoot, "test", "_output", "test-1.0.0.so"))
	assert.NoError(t, err)
	assert.NotNil(t, plugin)
}
