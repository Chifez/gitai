package cmd

import (
	"fmt"

	"github.com/Chifez/gitai/internal/config"
	"github.com/Chifez/gitai/pkg/ui"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage gitai configuration",
	Long:  "Get, set, list, or reset gitai configuration values.",
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "Print all config values and their sources",
	RunE: func(cmd *cobra.Command, args []string) error {
		entries, err := config.ListAll()
		if err != nil {
			return err
		}

		fmt.Println()
		fmt.Println(ui.Bold("gitai configuration:"))
		fmt.Println()
		for _, e := range entries {
			sourceLabel := ui.Magenta("(%s)", e.Source)
			fmt.Printf("  %-22s %s  %s\n", ui.Bold("%s", e.Key), e.Value, sourceLabel)
		}
		fmt.Println()
		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Print the resolved value of a config key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		val, source, err := config.GetValue(args[0])
		if err != nil {
			ui.Error("%v", err)
			return err
		}
		fmt.Printf("%s %s\n", val, ui.Magenta("(%s)", source))
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Update a config value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.SetValue(args[0], args[1]); err != nil {
			ui.Error("%v", err)
			return err
		}
		displayVal := args[1]
		if args[0] == "api_key" {
			if len(args[1]) > 4 {
				displayVal = args[1][:4] + "..."
			} else if args[1] != "" {
				displayVal = "***"
			}
		}
		ui.Success("Set %s = %s", args[0], displayVal)
		return nil
	},
}

var configResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset config to defaults",
	RunE: func(cmd *cobra.Command, args []string) error {
		return config.Reset()
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print the config file path",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := config.ConfigPath()
		if err != nil {
			return err
		}
		fmt.Println(p)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configResetCmd)
	configCmd.AddCommand(configPathCmd)
}
