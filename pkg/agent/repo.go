package agent

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/onosproject/onos-config-model-go/api/onos/configmodel"
	"github.com/onosproject/onos-config-model-go/pkg/compiler"
	"github.com/onosproject/onos-config-model-go/pkg/model"
	"github.com/onosproject/onos-config-model-go/pkg/repository"
	"github.com/onosproject/onos-lib-go/pkg/certs"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/northbound"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

var log = logging.GetLogger("config-model", "agent")

func getRepoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "repo",
	}
	cmd.AddCommand(getRepoServeCmd())
	cmd.AddCommand(getRepoGetCmd())
	cmd.AddCommand(getRepoListCmd())
	cmd.AddCommand(getRepoPushCmd())
	cmd.AddCommand(getRepoDeleteCmd())
	return cmd
}

func getRepoServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "serve",
		Short:        "Start the model repository server",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			caCert, _ := cmd.Flags().GetString("ca-cert")
			cert, _ := cmd.Flags().GetString("cert")
			key, _ := cmd.Flags().GetString("key")
			repoPath, _ := cmd.Flags().GetString("repo-path")
			if repoPath == "" {
				wd, err := os.Getwd()
				if err != nil {
					return err
				}
				repoPath = wd
			}
			buildPath, _ := cmd.Flags().GetString("build-path")
			if buildPath == "" {
				buildPath = filepath.Join(repoPath, "build")
			}
			port, _ := cmd.Flags().GetInt16("port")
			server := northbound.NewServer(&northbound.ServerConfig{
				CaPath:      &caCert,
				CertPath:    &cert,
				KeyPath:     &key,
				Port:        port,
				Insecure:    true,
				SecurityCfg: &northbound.SecurityConfig{},
			})
			repo := repository.NewRepository(repository.Config{
				Path: repoPath,
			})
			compiler := compiler.NewPluginCompiler(compiler.PluginCompilerConfig{
				TemplatePath: "pkg/compiler/templates",
				BuildPath:    buildPath,
				OutputPath:   repoPath,
			})
			server.AddService(repository.NewService(repo, compiler))

			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-c
				os.Exit(0)
			}()

			log.Infof("Starting repo server at '%s'", repoPath)
			err := server.Serve(func(address string) {
				log.Infof("Serving models at '%s' on %s", repoPath, address)
			})
			if err != nil {
				log.Errorf("Repo serve failed: %v", err)
				return err
			}
			return nil
		},
	}
	cmd.Flags().Int16P("port", "p", 5150, "the repository service port")
	cmd.Flags().String("repo-path", "", "the path in which to store the repository models")
	cmd.Flags().String("build-path", "", "the path in which to store temporary build artifacts")
	cmd.Flags().String("ca-cert", "", "the CA certificate")
	cmd.Flags().String("cert", "", "the certificate")
	cmd.Flags().String("key", "", "the key")
	return cmd
}

func getRepoGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "get",
		Short:        "Get a model from the repository",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			address, _ := cmd.Flags().GetString("address")
			name, _ := cmd.Flags().GetString("name")
			version, _ := cmd.Flags().GetString("version")
			conn, err := connect(address)
			if err != nil {
				return err
			}
			defer conn.Close()
			client := configmodel.NewRepositoryServiceClient(conn)
			request := &configmodel.GetModelRequest{
				Name:    name,
				Version: version,
			}
			ctx, cancel := newContext()
			defer cancel()
			response, err := client.GetModel(ctx, request)
			if err != nil {
				return err
			}
			var moduleInfos []model.ConfigModuleInfo
			for _, module := range response.Model.Modules {
				moduleInfos = append(moduleInfos, model.ConfigModuleInfo{
					Name:         model.Name(module.Name),
					Organization: module.Organization,
					Version:      model.Version(module.Version),
					Data:         module.Data,
				})
			}
			modelInfo := model.ConfigModelInfo{
				Name:    model.Name(response.Model.Name),
				Version: model.Version(response.Model.Version),
				Modules: moduleInfos,
				Plugin: model.ConfigPluginInfo{
					Name:    model.Name(response.Model.Name),
					Version: model.Version(response.Model.Version),
					File:    fmt.Sprintf("%s-%s.so", response.Model.Name, response.Model.Version),
				},
			}
			bytes, err := json.MarshalIndent(modelInfo, "", "  ")
			if err != nil {
				return err
			}
			println(string(bytes))
			return nil
		},
	}
	cmd.Flags().StringP("address", "a", "localhost:5150", "the repository address")
	cmd.Flags().StringP("name", "n", "", "the model name")
	cmd.Flags().StringP("version", "v", "", "the model version")
	return cmd
}

func getRepoListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "list",
		Short:        "List models in the repository",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			address, _ := cmd.Flags().GetString("address")
			conn, err := connect(address)
			if err != nil {
				return err
			}
			defer conn.Close()
			client := configmodel.NewRepositoryServiceClient(conn)
			request := &configmodel.ListModelsRequest{}
			ctx, cancel := newContext()
			defer cancel()
			response, err := client.ListModels(ctx, request)
			if err != nil {
				return err
			}
			for _, modelInfo := range response.Models {
				var moduleInfos []model.ConfigModuleInfo
				for _, module := range modelInfo.Modules {
					moduleInfos = append(moduleInfos, model.ConfigModuleInfo{
						Name:         model.Name(module.Name),
						Organization: module.Organization,
						Version:      model.Version(module.Version),
						Data:         module.Data,
					})
				}
				model := model.ConfigModelInfo{
					Name:    model.Name(modelInfo.Name),
					Version: model.Version(modelInfo.Version),
					Modules: moduleInfos,
					Plugin: model.ConfigPluginInfo{
						Name:    model.Name(modelInfo.Name),
						Version: model.Version(modelInfo.Version),
						File:    fmt.Sprintf("%s-%s.so", modelInfo.Name, modelInfo.Version),
					},
				}
				bytes, err := json.MarshalIndent(model, "", "  ")
				if err != nil {
					return err
				}
				println(string(bytes))
			}
			return nil
		},
	}
	cmd.Flags().StringP("address", "a", "localhost:5150", "the repository address")
	return cmd
}

func getRepoPushCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "push",
		Short:        "Push a model to the repository",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			address, _ := cmd.Flags().GetString("address")
			name, _ := cmd.Flags().GetString("name")
			version, _ := cmd.Flags().GetString("version")
			modules, _ := cmd.Flags().GetStringToString("module")
			conn, err := connect(address)
			if err != nil {
				return err
			}
			defer conn.Close()
			client := configmodel.NewRepositoryServiceClient(conn)
			model := &configmodel.ConfigModel{
				Name:    name,
				Version: version,
				Modules: []*configmodel.ConfigModule{},
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
				model.Modules = append(model.Modules, &configmodel.ConfigModule{
					Name:    name,
					Version: version,
					Data:    data,
				})
			}

			request := &configmodel.PushModelRequest{
				Model: model,
			}
			ctx, cancel := newContext()
			defer cancel()
			_, err = client.PushModel(ctx, request)
			return err
		},
	}
	cmd.Flags().StringP("address", "a", "localhost:5150", "the repository address")
	cmd.Flags().StringP("name", "n", "", "the model name")
	cmd.Flags().StringP("version", "v", "", "the model version")
	cmd.Flags().StringToStringP("module", "m", map[string]string{}, "model files")
	return cmd
}

func getRepoDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "delete",
		Short:        "Delete a model from the repository",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			address, _ := cmd.Flags().GetString("address")
			name, _ := cmd.Flags().GetString("name")
			version, _ := cmd.Flags().GetString("version")
			conn, err := connect(address)
			if err != nil {
				return err
			}
			defer conn.Close()
			client := configmodel.NewRepositoryServiceClient(conn)
			request := &configmodel.DeleteModelRequest{
				Name:    name,
				Version: version,
			}
			ctx, cancel := newContext()
			defer cancel()
			_, err = client.DeleteModel(ctx, request)
			return err
		},
	}
	cmd.Flags().StringP("address", "a", "localhost:5150", "the repository address")
	cmd.Flags().StringP("name", "n", "", "the model name")
	cmd.Flags().StringP("version", "v", "", "the model version")
	return cmd
}

func connect(address string) (*grpc.ClientConn, error) {
	cert, err := tls.X509KeyPair([]byte(certs.DefaultClientCrt), []byte(certs.DefaultClientKey))
	if err != nil {
		return nil, err
	}
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}

	// Connect to the first matching service
	return grpc.Dial(address, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
}

func newContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
	}()
	return ctx, cancel
}
