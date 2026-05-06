package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	SshcDirName = "sshc.d"
	IncludeLine = "Include sshc.d/*"
)

type Manager struct {
	SshDir string
	DryRun bool
}

func NewManager() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	return &Manager{
		SshDir: filepath.Join(home, ".ssh"),
	}, nil
}

func (m *Manager) SetDryRun(dryRun bool) {
	m.DryRun = dryRun
}

func (m *Manager) Init() error {
	sshcDir := filepath.Join(m.SshDir, SshcDirName)
	if m.DryRun {
		fmt.Printf("[Dry-run] Would create directory: %s\n", sshcDir)
	} else {
		if err := os.MkdirAll(sshcDir, 0700); err != nil {
			return fmt.Errorf("failed to create sshc directory: %w", err)
		}
	}

	configFile := filepath.Join(m.SshDir, "config")
	backupFile := configFile + ".backup"

	if _, err := os.Stat(configFile); err == nil {
		// 1. backup current ssh config if backup doesn't exist
		if _, err := os.Stat(backupFile); os.IsNotExist(err) {
			if m.DryRun {
				fmt.Printf("[Dry-run] Would backup %s to %s\n", configFile, backupFile)
			} else {
				content, err := os.ReadFile(configFile)
				if err != nil {
					return fmt.Errorf("failed to read ssh config for backup: %w", err)
				}
				if err := m.atomicWriteFile(backupFile, content, 0600); err != nil {
					return fmt.Errorf("failed to create backup: %w", err)
				}
			}
		}

		// 2. Create a new ssh config with the include
		content, err := os.ReadFile(configFile)
		if err != nil {
			return fmt.Errorf("failed to read ssh config: %w", err)
		}
		// Check if Include line already exists
		found := false
		for line := range strings.SplitSeq(string(content), "\n") {
			if strings.TrimSpace(line) == IncludeLine {
				found = true
				break
			}
		}

		if !found {
			if m.DryRun {
				fmt.Printf("[Dry-run] Would prepend '%s' to %s\n", IncludeLine, configFile)
			} else {
				newContent := IncludeLine + "\n" + string(content)
				if err := m.atomicWriteFile(configFile, []byte(newContent), 0600); err != nil {
					return fmt.Errorf("failed to update ssh config: %w", err)
				}
			}
		}
	} else if os.IsNotExist(err) {
		// Create a new ssh config with the include
		if m.DryRun {
			fmt.Printf("[Dry-run] Would create %s with content: %s\n", configFile, IncludeLine)
		} else {
			if err := m.atomicWriteFile(configFile, []byte(IncludeLine+"\n"), 0600); err != nil {
				return fmt.Errorf("failed to create ssh config: %w", err)
			}
		}
	} else {
		return fmt.Errorf("failed to stat ssh config: %w", err)
	}

	return nil
}

func (m *Manager) GetConfigPath(name string) string {
	return filepath.Join(m.SshDir, SshcDirName, name)
}

func (m *Manager) AddConfig(name string, content string) error {
	if err := m.ValidateContent(content); err != nil {
		return err
	}

	// Extract host from content
	var host string
	for line := range strings.SplitSeq(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(trimmed), "host ") {
			host = strings.TrimSpace(trimmed[len("host "):])
			break
		}
	}

	if host != "" {
		if err := m.checkDuplicateHost(host, name); err != nil {
			return err
		}
	}

	configPath := m.GetConfigPath(name)
	if m.DryRun {
		fmt.Printf("[Dry-run] Would write config to %s:\n%s\n", configPath, content)
		return nil
	}
	if err := m.atomicWriteFile(configPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write config %s: %w", name, err)
	}
	return nil
}

func (m *Manager) atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, filepath.Base(path)+".tmp.*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		if err != nil {
			_ = os.Remove(tmpPath)
		}
	}()

	if err = tmpFile.Chmod(perm); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to chmod temp file: %w", err)
	}

	if _, err = tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	if err = tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err = os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file to target: %w", err)
	}

	return nil
}

func (m *Manager) RemoveConfig(name string) error {
	_, err := m.RemoveConfigWithKey(name, false)
	return err
}

func (m *Manager) RemoveConfigWithKey(name string, deleteKey bool) (string, error) {
	configPath := m.GetConfigPath(name)
	content, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("config %s does not exist", name)
		}
		return "", fmt.Errorf("failed to read config %s: %w", name, err)
	}

	var identityFile string
	for line := range strings.SplitSeq(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(trimmed), "identityfile ") {
			identityFile = strings.TrimSpace(trimmed[len("identityfile "):])
			break
		}
	}

	if identityFile != "" {
		_, errPriv := os.Stat(identityFile)
		_, errPub := os.Stat(identityFile + ".pub")
		if os.IsNotExist(errPriv) && os.IsNotExist(errPub) {
			identityFile = ""
		}
	}

	if deleteKey && identityFile != "" {
		if m.DryRun {
			fmt.Printf("[Dry-run] Would remove private key: %s\n", identityFile)
			fmt.Printf("[Dry-run] Would remove public key: %s.pub\n", identityFile)
		} else {
			// Try to remove the private key
			_ = os.Remove(identityFile)
			// Try to remove the public key
			_ = os.Remove(identityFile + ".pub")
		}
	}

	if m.DryRun {
		fmt.Printf("[Dry-run] Would remove config file: %s\n", configPath)
		return identityFile, nil
	}

	if err := os.Remove(configPath); err != nil {
		return "", fmt.Errorf("failed to remove config %s: %w", name, err)
	}

	return identityFile, nil
}

type ConfigOptions struct {
	Host         string
	Hostname     string
	User         string
	Port         int
	IdentityFile string
	ForwardAgent string // "yes", "no", or ""
	ProxyJump    string
}

func (opts ConfigOptions) Validate() error {
	var errs []string
	if opts.Host == "" {
		errs = append(errs, "mandatory field 'Host' (alias) is missing")
	}
	if opts.Hostname == "" {
		errs = append(errs, "mandatory field 'Hostname' (address) is missing")
	}
	if opts.ForwardAgent != "" && opts.ForwardAgent != "yes" && opts.ForwardAgent != "no" {
		errs = append(errs, "invalid value for 'ForwardAgent': must be 'yes' or 'no'")
	}
	if opts.Port < 0 || opts.Port > 65535 {
		errs = append(errs, fmt.Sprintf("invalid port: %d", opts.Port))
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

func (m *Manager) ValidateContent(content string) error {
	lines := strings.Split(content, "\n")
	var host, hostname string
	var errs []string

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		parts := strings.Fields(trimmed)
		if len(parts) < 2 {
			// Some options might have no value which is generally invalid in SSH config
			// but we mostly care about our mandatory ones
			continue
		}

		key := strings.ToLower(parts[0])
		value := strings.Join(parts[1:], " ")

		switch key {
		case "host":
			host = value
		case "hostname":
			hostname = value
		case "forwardagent":
			if value != "yes" && value != "no" {
				errs = append(errs, fmt.Sprintf("line %d: invalid value for 'ForwardAgent': %s (must be 'yes' or 'no')", i+1, value))
			}
		case "port":
			var p int
			if _, err := fmt.Sscanf(value, "%d", &p); err != nil || p < 0 || p > 65535 {
				errs = append(errs, fmt.Sprintf("line %d: invalid port: %s", i+1, value))
			}
		}
	}

	if host == "" {
		errs = append(errs, "missing mandatory field 'Host'")
	}
	if hostname == "" {
		errs = append(errs, "missing mandatory field 'Hostname'")
	}

	if len(errs) > 0 {
		return fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

func (m *Manager) UpdateConfig(name string, opts ConfigOptions) error {
	path := m.GetConfigPath(name)
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if opts.Host != "" {
		if err := m.checkDuplicateHost(opts.Host, name); err != nil {
			return err
		}
	}

	// For UpdateConfig, we should validate the resulting content
	// but we can also pre-validate the options if they are provided.
	// We'll validate the final content at the end of this function.

	lines := strings.Split(string(content), "\n")
	newLines := make([]string, 0, len(lines))

	foundFields := make(map[string]bool)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lowerTrimmed := strings.ToLower(trimmed)

		if strings.HasPrefix(lowerTrimmed, "host ") && opts.Host != "" {
			newLines = append(newLines, "Host "+quoteIfSpace(opts.Host))
			foundFields["host"] = true
		} else if strings.HasPrefix(lowerTrimmed, "hostname ") {
			foundFields["hostname"] = true
			if opts.Hostname != "" {
				indent := line[:strings.Index(strings.ToLower(line), "hostname")]
				newLines = append(newLines, indent+"Hostname "+quoteIfSpace(opts.Hostname))
			} else {
				newLines = append(newLines, line)
			}
		} else if strings.HasPrefix(lowerTrimmed, "user ") {
			foundFields["user"] = true
			if opts.User != "" {
				indent := line[:strings.Index(strings.ToLower(line), "user")]
				newLines = append(newLines, indent+"User "+quoteIfSpace(opts.User))
			} else {
				newLines = append(newLines, line)
			}
		} else if strings.HasPrefix(lowerTrimmed, "port ") {
			foundFields["port"] = true
			if opts.Port != 0 {
				indent := line[:strings.Index(strings.ToLower(line), "port")]
				newLines = append(newLines, indent+fmt.Sprintf("Port %d", opts.Port))
			} else {
				newLines = append(newLines, line)
			}
		} else if strings.HasPrefix(lowerTrimmed, "identityfile ") {
			foundFields["identityfile"] = true
			if opts.IdentityFile != "" {
				identity := opts.IdentityFile
				// Ensure path is absolute for SSH config
				if absPath, err := filepath.Abs(identity); err == nil {
					identity = absPath
				}
				indent := line[:strings.Index(strings.ToLower(line), "identityfile")]
				newLines = append(newLines, indent+"IdentityFile "+quoteIfSpace(identity))
			} else {
				newLines = append(newLines, line)
			}
		} else if strings.HasPrefix(lowerTrimmed, "forwardagent ") {
			foundFields["forwardagent"] = true
			if opts.ForwardAgent != "" {
				indent := line[:strings.Index(strings.ToLower(line), "forwardagent")]
				newLines = append(newLines, indent+"ForwardAgent "+opts.ForwardAgent)
			} else {
				newLines = append(newLines, line)
			}
		} else if strings.HasPrefix(lowerTrimmed, "proxyjump ") {
			foundFields["proxyjump"] = true
			if opts.ProxyJump != "" {
				indent := line[:strings.Index(strings.ToLower(line), "proxyjump")]
				newLines = append(newLines, indent+"ProxyJump "+quoteIfSpace(opts.ProxyJump))
			} else {
				newLines = append(newLines, line)
			}
		} else {
			if trimmed != "" || line != "" {
				newLines = append(newLines, line)
			}
		}
	}

	// Add fields if they were not present but are now requested
	if !foundFields["user"] && opts.User != "" {
		newLines = append(newLines, "    User "+quoteIfSpace(opts.User))
	}
	if !foundFields["hostname"] && opts.Hostname != "" {
		newLines = append(newLines, "    Hostname "+quoteIfSpace(opts.Hostname))
	}
	if !foundFields["port"] && opts.Port != 0 {
		newLines = append(newLines, fmt.Sprintf("    Port %d", opts.Port))
	}
	if !foundFields["identityfile"] && opts.IdentityFile != "" {
		identity := opts.IdentityFile
		if absPath, err := filepath.Abs(identity); err == nil {
			identity = absPath
		}
		newLines = append(newLines, "    IdentityFile "+quoteIfSpace(identity))
	}
	if !foundFields["forwardagent"] && opts.ForwardAgent != "" {
		newLines = append(newLines, "    ForwardAgent "+opts.ForwardAgent)
	}
	if !foundFields["proxyjump"] && opts.ProxyJump != "" {
		newLines = append(newLines, "    ProxyJump "+quoteIfSpace(opts.ProxyJump))
	}

	updatedContent := strings.Join(newLines, "\n")
	if err := m.ValidateContent(updatedContent); err != nil {
		return err
	}

	if m.DryRun {
		fmt.Printf("[Dry-run] Would update config at %s with:\n%s\n", path, updatedContent)
		return nil
	}

	return m.atomicWriteFile(path, []byte(updatedContent), 0600)
}

func (opts ConfigOptions) String() string {
	var sb strings.Builder
	if opts.Host != "" {
		sb.WriteString("Host " + quoteIfSpace(opts.Host) + "\n")
	}
	if opts.Hostname != "" {
		sb.WriteString("    Hostname " + quoteIfSpace(opts.Hostname) + "\n")
	}
	if opts.User != "" {
		sb.WriteString("    User " + quoteIfSpace(opts.User) + "\n")
	}
	if opts.Port != 0 {
		sb.WriteString(fmt.Sprintf("    Port %d\n", opts.Port))
	}
	if opts.IdentityFile != "" {
		identity := opts.IdentityFile
		if absPath, err := filepath.Abs(identity); err == nil {
			identity = absPath
		}
		sb.WriteString("    IdentityFile " + quoteIfSpace(identity) + "\n")
	}
	if opts.ForwardAgent != "" {
		sb.WriteString("    ForwardAgent " + opts.ForwardAgent + "\n")
	}
	if opts.ProxyJump != "" {
		sb.WriteString("    ProxyJump " + quoteIfSpace(opts.ProxyJump) + "\n")
	}
	return sb.String()
}

func quoteIfSpace(s string) string {
	if strings.Contains(s, " ") {
		return "\"" + s + "\""
	}
	return s
}

func (m *Manager) ListConfigs() ([]string, error) {
	sshcDir := filepath.Join(m.SshDir, SshcDirName)
	entries, err := os.ReadDir(sshcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read sshc directory: %w", err)
	}

	var configs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			configs = append(configs, entry.Name())
		}
	}
	slices.Sort(configs)
	return configs, nil
}

func (m *Manager) checkDuplicateHost(host, currentConfigName string) error {
	configs, err := m.ListConfigs()
	if err != nil {
		return err
	}

	for _, name := range configs {
		if name == currentConfigName {
			continue
		}

		content, err := os.ReadFile(m.GetConfigPath(name))
		if err != nil {
			continue
		}

		for line := range strings.SplitSeq(string(content), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(strings.ToLower(trimmed), "host ") {
				existingHost := strings.TrimSpace(trimmed[len("host "):])
				if existingHost == host {
					return fmt.Errorf("host alias '%s' is already defined in config '%s'", host, name)
				}
			}
		}
	}
	return nil
}
