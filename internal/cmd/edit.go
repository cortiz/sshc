package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

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
			opts := config.ConfigOptions{
				Host:         editHost,
				Hostname:     editHostname,
				User:         editUser,
				Port:         editPort,
				IdentityFile: editIdentity,
				ForwardAgent: editForwardAgent,
				ProxyJump:    editProxyJump,
			}
			return m.UpdateConfig(name, opts)
		}

		return openEditor(configPath)
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

	cmd := exec.Command(editor, path)
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
