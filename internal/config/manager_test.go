package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestManager_Init(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sshc-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	m := &Manager{
		SshDir: tmpDir,
	}

	if err := m.Init(); err != nil {
		t.Errorf("Init() error = %v", err)
	}

	// Check if sshc.d exists
	if _, err := os.Stat(filepath.Join(tmpDir, SshcDirName)); os.IsNotExist(err) {
		t.Errorf("sshc.d directory not created")
	}

	// Check if config file has Include line
	content, err := os.ReadFile(filepath.Join(tmpDir, "config"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), IncludeLine) {
		t.Errorf("config file does not contain Include line")
	}
}

func TestManager_Init_Backup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sshc-test-backup")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	m := &Manager{
		SshDir: tmpDir,
	}

	// Create initial config
	initialContent := "Host initial\n  Hostname example.com"
	if err := os.WriteFile(filepath.Join(tmpDir, "config"), []byte(initialContent), 0600); err != nil {
		t.Fatal(err)
	}

	if err := m.Init(); err != nil {
		t.Errorf("Init() error = %v", err)
	}

	// Check if backup exists and has initial content
	backupContent, err := os.ReadFile(filepath.Join(tmpDir, "config.backup"))
	if err != nil {
		t.Fatalf("Failed to read backup file: %v", err)
	}
	if string(backupContent) != initialContent {
		t.Errorf("Backup content mismatch. Got: %s, Want: %s", string(backupContent), initialContent)
	}

	// Check if config file has Include line prepended
	configContent, err := os.ReadFile(filepath.Join(tmpDir, "config"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(configContent), IncludeLine) {
		t.Errorf("config file does not have Include line prepended. Got: %s", string(configContent))
	}

	// Run Init again, should be idempotent and not prepend again
	if err := m.Init(); err != nil {
		t.Errorf("Second Init() error = %v", err)
	}

	configContentSecond, err := os.ReadFile(filepath.Join(tmpDir, "config"))
	if err != nil {
		t.Fatal(err)
	}
	count := strings.Count(string(configContentSecond), IncludeLine)
	if count != 1 {
		t.Errorf("Include line found %d times, expected 1", count)
	}

	// Verify that backup was NOT overwritten
	// 1. Change the config content
	newConfigContent := "Host new\n  Hostname updated.com"
	if err := os.WriteFile(filepath.Join(tmpDir, "config"), []byte(newConfigContent), 0600); err != nil {
		t.Fatal(err)
	}

	// 2. Run Init again
	if err := m.Init(); err != nil {
		t.Fatal(err)
	}

	// 3. Backup should still have the ORIGINAL content, not the newConfigContent
	backupContentAfter, err := os.ReadFile(filepath.Join(tmpDir, "config.backup"))
	if err != nil {
		t.Fatal(err)
	}
	if string(backupContentAfter) != initialContent {
		t.Errorf("Backup was overwritten! Got: %s, Want: %s", string(backupContentAfter), initialContent)
	}
}

func TestManager_AddRemoveConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sshc-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	m := &Manager{
		SshDir: tmpDir,
	}
	_ = m.Init()

	name := "test-config"
	content := "Host test\n  Hostname example.com"

	if err := m.AddConfig(name, content); err != nil {
		t.Errorf("AddConfig() error = %v", err)
	}

	configs, err := m.ListConfigs()
	if err != nil {
		t.Errorf("ListConfigs() error = %v", err)
	}
	if !slices.Contains(configs, name) {
		t.Errorf("Config %s not found in list", name)
	}

	if err := m.RemoveConfig(name); err != nil {
		t.Errorf("RemoveConfig() error = %v", err)
	}

	configs, _ = m.ListConfigs()
	if len(configs) != 0 {
		t.Errorf("Expected 0 configs after removal, got %d", len(configs))
	}
}

func TestManager_RemoveConfig_WithKey(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sshc-test-rm-key")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	m := &Manager{
		SshDir: tmpDir,
	}
	_ = m.Init()

	name := "key-config"
	keyPath := filepath.Join(tmpDir, "test-key")
	content := "Host test\n  IdentityFile " + keyPath

	// Create dummy key files
	if err := os.WriteFile(keyPath, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath+".pub", []byte("public"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := m.AddConfig(name, content); err != nil {
		t.Fatal(err)
	}

	// 1. Remove WITHOUT deleting key
	idFile, err := m.RemoveConfigWithKey(name, false)
	if err != nil {
		t.Errorf("RemoveConfigWithKey(false) error = %v", err)
	}
	if idFile != keyPath {
		t.Errorf("Expected idFile %s, got %s", keyPath, idFile)
	}

	// Verify key still exists
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Errorf("Key should still exist")
	}

	// 2. Add back and remove WITH deleting key
	if err := m.AddConfig(name, content); err != nil {
		t.Fatal(err)
	}
	idFile, err = m.RemoveConfigWithKey(name, true)
	if err != nil {
		t.Errorf("RemoveConfigWithKey(true) error = %v", err)
	}
	if idFile != keyPath {
		t.Errorf("Expected idFile %s, got %s", keyPath, idFile)
	}

	// Verify key is deleted
	if _, err := os.Stat(keyPath); !os.IsNotExist(err) {
		t.Errorf("Key should be deleted")
	}
	if _, err := os.Stat(keyPath + ".pub"); !os.IsNotExist(err) {
		t.Errorf("Public key should be deleted")
	}
}

func TestManager_ListConfigs_Sorted(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sshc-test-list")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	m := &Manager{
		SshDir: tmpDir,
	}
	_ = m.Init()

	// Add configs in non-alphabetical order
	names := []string{"zebra", "apple", "banana"}
	for _, name := range names {
		if err := m.AddConfig(name, "Host "+name); err != nil {
			t.Fatal(err)
		}
	}

	configs, err := m.ListConfigs()
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"apple", "banana", "zebra"}
	if !slices.Equal(configs, expected) {
		t.Errorf("ListConfigs() not sorted. Got: %v, Want: %v", configs, expected)
	}
}

func TestManager_UpdateConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sshc-test-update")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	m := &Manager{
		SshDir: tmpDir,
	}
	_ = m.Init()

	name := "update-test"
	initialContent := "Host old-host\n    User old-user\n"
	if err := m.AddConfig(name, initialContent); err != nil {
		t.Fatal(err)
	}

	// Update host and user
	opts := ConfigOptions{
		Host: "new-host",
		User: "new-user",
	}
	if err := m.UpdateConfig(name, opts); err != nil {
		t.Errorf("UpdateConfig() error = %v", err)
	}

	content, err := os.ReadFile(m.GetConfigPath(name))
	if err != nil {
		t.Fatal(err)
	}

	res := string(content)
	if !strings.Contains(res, "Host new-host") {
		t.Errorf("Expected 'Host new-host', got: %s", res)
	}
	if !strings.Contains(res, "User new-user") {
		t.Errorf("Expected 'User new-user', got: %s", res)
	}
	if strings.Contains(res, "old-host") || strings.Contains(res, "old-user") {
		t.Errorf("Old content still present: %s", res)
	}

	// Add IdentityFile
	opts = ConfigOptions{
		IdentityFile: "/path/to/key",
	}
	if err := m.UpdateConfig(name, opts); err != nil {
		t.Errorf("UpdateConfig() error = %v", err)
	}

	content, err = os.ReadFile(m.GetConfigPath(name))
	if err != nil {
		t.Fatal(err)
	}
	res = string(content)
	if !strings.Contains(res, "IdentityFile") {
		t.Errorf("IdentityFile not added")
	}

	// Add more options
	opts = ConfigOptions{
		Hostname:     "example.com",
		Port:         2222,
		ForwardAgent: "yes",
		ProxyJump:    "jump-host",
	}
	if err := m.UpdateConfig(name, opts); err != nil {
		t.Errorf("UpdateConfig() error = %v", err)
	}

	content, err = os.ReadFile(m.GetConfigPath(name))
	if err != nil {
		t.Fatal(err)
	}
	res = string(content)
	if !strings.Contains(res, "Hostname example.com") {
		t.Errorf("Hostname not added")
	}
	if !strings.Contains(res, "Port 2222") {
		t.Errorf("Port not added")
	}
	if !strings.Contains(res, "ForwardAgent yes") {
		t.Errorf("ForwardAgent not added")
	}
	if !strings.Contains(res, "ProxyJump jump-host") {
		t.Errorf("ProxyJump not added")
	}
}

func TestManager_UpdateConfig_Indentation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sshc-test-indent")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	m := &Manager{
		SshDir: tmpDir,
	}
	_ = m.Init()

	name := "indent-test"
	// Use 8 spaces for indentation
	initialContent := "Host my-host\n        Hostname old-hostname\n"
	if err := m.AddConfig(name, initialContent); err != nil {
		t.Fatal(err)
	}

	opts := ConfigOptions{
		Hostname: "new-hostname",
	}
	if err := m.UpdateConfig(name, opts); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(m.GetConfigPath(name))
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Hostname") {
			if !strings.HasPrefix(line, "        ") {
				t.Errorf("Indentation lost. Expected 8 spaces at start of line: %q", line)
			}
		}
	}
}
