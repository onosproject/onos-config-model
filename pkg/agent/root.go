package agent

import "github.com/spf13/cobra"

func GetRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config-agent",
		Short: "Config model/plugin management agent",
	}
	cmd.AddCommand(getPluginCmd())
	cmd.AddCommand(getRepoCmd())
	return cmd
}
