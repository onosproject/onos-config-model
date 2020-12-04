package agent

import (
	"errors"
	"fmt"
	"github.com/onosproject/onos-config-model-go/pkg/compiler"
	"github.com/onosproject/onos-config-model-go/pkg/model"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func getPluginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "plugin",
	}
	cmd.AddCommand(getPluginCompileCmd())
	return cmd
}

func getPluginCompileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "compile",
		Short:        "Compile a config model plugin",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			version, _ := cmd.Flags().GetString("version")
			modules, _ := cmd.Flags().GetStringToString("module")

			outputPath, _ := cmd.Flags().GetString("output-path")
			if outputPath == "" {
				dir, err := os.Getwd()
				if err != nil {
					return err
				}
				outputPath = dir
			}
			buildPath, _ := cmd.Flags().GetString("build-path")
			if buildPath == "" {
				buildPath = filepath.Join(outputPath, "build")
			}

			modelInfo := model.ConfigModelInfo{
				Name:    model.Name(name),
				Version: model.Version(version),
				Modules: []model.ConfigModuleInfo{},
				Plugin: model.ConfigPluginInfo{
					Name:    model.Name(name),
					Version: model.Version(version),
					File:    fmt.Sprintf("%s-%s.so", name, version),
				},
			}
			for nameVersion, module := range modules {
				names := strings.Split(nameVersion, "@")
				if len(names) != 2 {
					return errors.New("module name must be in the format $name@$version")
				}
				name, version := names[0], names[1]
				data, err := ioutil.ReadFile(module)
				if err != nil {
					return err
				}
				modelInfo.Modules = append(modelInfo.Modules, model.ConfigModuleInfo{
					Name:    model.Name(name),
					Version: model.Version(version),
					Data:    data,
				})
			}

			config := compiler.PluginCompilerConfig{
				TemplatePath: "pkg/compiler/templates",
				OutputPath:   outputPath,
			}
			return compiler.CompilePlugin(modelInfo, config)
		},
	}
	cmd.Flags().StringP("name", "n", "", "the model name")
	cmd.Flags().StringP("version", "v", "", "the model version")
	cmd.Flags().StringToStringP("module", "m", map[string]string{}, "model files")
	cmd.Flags().StringP("build-path", "b", "", "the build path")
	cmd.Flags().StringP("output-path", "o", "", "the output path")
	return cmd
}
