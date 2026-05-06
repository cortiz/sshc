package cmd

import (
	"fmt"

	"sshc/internal/config"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List managed SSH configurations",
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := config.NewManager()
		if err != nil {
			return err
		}

		configs, err := m.ListConfigs()
		if err != nil {
			return err
		}

		if len(configs) == 0 {
			fmt.Println("No managed configs found.")
			return nil
		}

		fmt.Println("Managed SSH configs:")
		for _, cfg := range configs {
			fmt.Printf("  - %s\n", cfg)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
