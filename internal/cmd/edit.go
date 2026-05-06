package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"sshc/internal/config"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	editHost         string
	editHostname     string
	editUser         string
	editPort         int
	editIdentity     string
	editForwardAgent string
	editProxyJump    string
)

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit SSH configuration",
}

var editConfigCmd = &cobra.Command{
	Use:   "config NAME",
	Short: "Edit an SSH config entry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		m, err := config.NewManager()
		if err != nil {
			return err
		}
		m.SetDryRun(dryRun)

		configPath := m.GetConfigPath(name)
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return fmt.Errorf("config %s does not exist", name)
		}

		// Check if any flag was provided
		hasFlag := false
		cmd.Flags().Visit(func(f *pflag.Flag) {
			hasFlag = true
		})

		if hasFlag {
			if editIdentity != "" {
				if _, err := os.Stat(editIdentity); os.IsNotExist(err) {
					fmt.Printf("Warning: IdentityFile %s does not exist\n", editIdentity)
				}
			}
			opts := config.ConfigOptions{
				Host:         editHost,
				Hostname:     editHostname,
				User:         editUser,
				Port:         editPort,
				IdentityFile: editIdentity,
				ForwardAgent: editForwardAgent,
				ProxyJump:    editProxyJump,
			}
			if err := m.UpdateConfig(name, opts); err != nil {
				return err
			}
			if dryRun {
				fmt.Printf("Config %s would be updated successfully (dry-run)\n", name)
			} else {
				fmt.Printf("Config %s updated successfully\n", name)
			}
			return nil
		}

		if dryRun {
			fmt.Printf("[Dry-run] Would open editor for %s\n", configPath)
			return nil
		}

		if err := openEditor(configPath); err != nil {
			return err
		}

		// Re-read and validate after editing
		newContent, err := os.ReadFile(configPath)
		if err != nil {
			return err
		}
		return m.ValidateContent(string(newContent))
	},
}

func openEditor(path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		if runtime.GOOS == "windows" {
			editor = "notepad"
		} else {
			editor = "vi"
		}
	}

	// Split editor if it contains arguments
	editorParts := strings.Fields(editor)
	if len(editorParts) == 0 {
		return fmt.Errorf("invalid EDITOR environment variable")
	}

	args := append(editorParts[1:], path)
	cmd := exec.Command(editorParts[0], args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func init() {
	editConfigCmd.Flags().StringVar(&editHost, "host", "", "SSH Host alias")
	editConfigCmd.Flags().StringVar(&editHostname, "hostname", "", "SSH Hostname (real address)")
	editConfigCmd.Flags().StringVar(&editUser, "user", "", "SSH User")
	editConfigCmd.Flags().IntVar(&editPort, "port", 0, "SSH Port")
	editConfigCmd.Flags().StringVar(&editIdentity, "identity", "", "Path to identity file")
	editConfigCmd.Flags().StringVar(&editForwardAgent, "forward-agent", "", "Forward SSH Agent (yes/no)")
	editConfigCmd.Flags().StringVar(&editProxyJump, "proxy-jump", "", "SSH ProxyJump")

	editCmd.AddCommand(editConfigCmd)
	rootCmd.AddCommand(editCmd)
}
