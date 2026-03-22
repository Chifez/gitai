// Package config handles configuration loading, writing, priority resolution,
// and the first-run setup wizard.
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Chifez/gitai/pkg/provider"
	"github.com/Chifez/gitai/pkg/provider/openai"
	"github.com/Chifez/gitai/pkg/ui"
)

const (
	configDirName  = ".gitai"
	configFileName = "config.yaml"
)

// Config holds all gitai configuration values.
type Config struct {
	Provider          string `yaml:"provider"`
	APIKey            string `yaml:"api_key,omitempty"`
	Model             string `yaml:"model"`
	Style             string `yaml:"style"`
	AutoPush          bool   `yaml:"auto_push"`
	MaxLength         int    `yaml:"max_length"`
	Lang              string `yaml:"lang"`
	IncludeBody       bool   `yaml:"include_body"`
	DefaultRemoteName string `yaml:"default_remote_name"`
	AutoSetUpstream   bool   `yaml:"auto_set_upstream"`
}

// Defaults returns a Config with all built-in default values.
func Defaults() Config {
	return Config{
		Provider:          "openai",
		Model:             "gpt-4o-mini",
		Style:             "conventional",
		AutoPush:          true,
		MaxLength:         72,
		Lang:              "english",
		IncludeBody:       true,
		DefaultRemoteName: "origin",
		AutoSetUpstream:   true,
	}
}

// ConfigPath returns the full path to the config file.
func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	return filepath.Join(home, configDirName, configFileName), nil
}

// Load reads the config file and returns a Config with priority resolution applied.
// Priority: CLI flags > env vars > config file > defaults.
// The flagOverrides map contains any values passed via CLI flags.
func Load(flagOverrides map[string]string) (*Config, error) {
	cfg := Defaults()

	// Load config file
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err == nil {
		// Config file exists, parse it
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}
	// If file doesn't exist, we just use defaults — not an error

	// Apply environment variable overrides
	applyEnvOverrides(&cfg)

	// Apply CLI flag overrides
	applyFlagOverrides(&cfg, flagOverrides)

	return &cfg, nil
}

// applyEnvOverrides applies environment variable values to the config.
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("OPENAI_API_KEY"); v != "" {
		cfg.APIKey = v
	}
	if v := os.Getenv("GITAI_MODEL"); v != "" {
		cfg.Model = v
	}
	if v := os.Getenv("GITAI_PROVIDER"); v != "" {
		cfg.Provider = v
	}
	if v := os.Getenv("GITAI_STYLE"); v != "" {
		cfg.Style = v
	}
	if v := os.Getenv("GITAI_AUTO_PUSH"); v != "" {
		cfg.AutoPush = strings.ToLower(v) == "true"
	}
	if v := os.Getenv("GITAI_LANG"); v != "" {
		cfg.Lang = v
	}
}

// applyFlagOverrides applies CLI flag values to the config.
func applyFlagOverrides(cfg *Config, flags map[string]string) {
	if flags == nil {
		return
	}
	if v, ok := flags["provider"]; ok && v != "" {
		cfg.Provider = v
	}
	if v, ok := flags["api_key"]; ok && v != "" {
		cfg.APIKey = v
	}
	if v, ok := flags["model"]; ok && v != "" {
		cfg.Model = v
	}
	if v, ok := flags["style"]; ok && v != "" {
		cfg.Style = v
	}
	if v, ok := flags["max_length"]; ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.MaxLength = n
		}
	}
	if v, ok := flags["lang"]; ok && v != "" {
		cfg.Lang = v
	}
}

// Save writes the config to disk atomically.
// Writes to a temp file first, then renames — crash-safe.
func Save(cfg *Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	// Ensure directory exists with 0700 permissions
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Atomic write: temp file → rename
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("failed to write temp config: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp) // clean up
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// SetValue updates a single key in the config file.
func SetValue(key, value string) error {
	cfg, err := Load(nil)
	if err != nil {
		return err
	}

	switch key {
	case "provider":
		cfg.Provider = value
	case "api_key":
		cfg.APIKey = value
	case "model":
		cfg.Model = value
	case "style":
		cfg.Style = value
	case "auto_push":
		cfg.AutoPush = strings.ToLower(value) == "true"
	case "max_length":
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("max_length must be a number: %w", err)
		}
		cfg.MaxLength = n
	case "lang":
		cfg.Lang = value
	case "include_body":
		cfg.IncludeBody = strings.ToLower(value) == "true"
	case "default_remote_name":
		cfg.DefaultRemoteName = value
	case "auto_set_upstream":
		cfg.AutoSetUpstream = strings.ToLower(value) == "true"
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	return Save(cfg)
}

// GetValue returns the resolved value of a single config key.
func GetValue(key string) (string, string, error) {
	cfg, err := Load(nil)
	if err != nil {
		return "", "", err
	}

	// Determine source
	source := resolveSource(key)

	var val string
	switch key {
	case "provider":
		val = cfg.Provider
	case "api_key":
		if len(cfg.APIKey) > 4 {
			val = cfg.APIKey[:4] + "..." // mask API key
		} else if cfg.APIKey != "" {
			val = "***"
		}
	case "model":
		val = cfg.Model
	case "style":
		val = cfg.Style
	case "auto_push":
		val = strconv.FormatBool(cfg.AutoPush)
	case "max_length":
		val = strconv.Itoa(cfg.MaxLength)
	case "lang":
		val = cfg.Lang
	case "include_body":
		val = strconv.FormatBool(cfg.IncludeBody)
	case "default_remote_name":
		val = cfg.DefaultRemoteName
	case "auto_set_upstream":
		val = strconv.FormatBool(cfg.AutoSetUpstream)
	default:
		return "", "", fmt.Errorf("unknown config key: %s", key)
	}

	return val, source, nil
}

// ListAll returns all config keys with values and sources.
func ListAll() ([]ConfigEntry, error) {
	cfg, err := Load(nil)
	if err != nil {
		return nil, err
	}

	keys := []string{
		"provider", "api_key", "model", "style", "auto_push",
		"max_length", "lang", "include_body", "default_remote_name", "auto_set_upstream",
	}

	var entries []ConfigEntry
	for _, key := range keys {
		val, source, _ := GetValue(key)
		_ = cfg // suppress unused
		entries = append(entries, ConfigEntry{Key: key, Value: val, Source: source})
	}

	return entries, nil
}

// ConfigEntry represents a single config key-value pair with its source.
type ConfigEntry struct {
	Key    string
	Value  string
	Source string // "flag", "env", "file", "default"
}

// resolveSource determines where a config value came from.
func resolveSource(key string) string {
	envMap := map[string]string{
		"api_key":  "OPENAI_API_KEY",
		"model":    "GITAI_MODEL",
		"provider": "GITAI_PROVIDER",
		"style":    "GITAI_STYLE",
		"auto_push": "GITAI_AUTO_PUSH",
		"lang":     "GITAI_LANG",
	}

	if envVar, ok := envMap[key]; ok {
		if os.Getenv(envVar) != "" {
			return "env"
		}
	}

	// Check if config file exists and has this key
	path, err := ConfigPath()
	if err == nil {
		if _, err := os.Stat(path); err == nil {
			return "file"
		}
	}

	return "default"
}

// Reset resets the config to defaults after user confirmation.
func Reset() error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Reset config to defaults? This cannot be undone. (y/n) ")
	input, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(input)) != "y" {
		ui.Info("Reset cancelled.")
		return nil
	}

	cfg := Defaults()
	if err := Save(&cfg); err != nil {
		return err
	}

	ui.Success("Config reset to defaults.")
	return nil
}

// BuildProvider creates a Provider instance based on the current config.
func (c *Config) BuildProvider() (provider.Provider, error) {
	switch c.Provider {
	case "openai":
		if c.APIKey == "" {
			return nil, fmt.Errorf("OpenAI API key not found. Set OPENAI_API_KEY or run: gitai config set api_key YOUR_KEY")
		}
		return openai.New(c.APIKey, c.Model), nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", c.Provider)
	}
}

// RunSetupWizard runs the first-time setup wizard.
func RunSetupWizard() (*Config, error) {
	// Clean up temp config if interrupted
	path, _ := ConfigPath()
	defer os.Remove(path + ".tmp")

	ui.Info("No config found. Running first-time setup...")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	cfg := Defaults()

	// API Key
	fmt.Print(ui.Bold("Enter your OpenAI API key: "))
	apiKey, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	cfg.APIKey = strings.TrimSpace(apiKey)

	// Model
	fmt.Printf("Default model %s: ", ui.Dim("[%s]", cfg.Model))
	model, _ := reader.ReadString('\n')
	if m := strings.TrimSpace(model); m != "" {
		cfg.Model = m
	}

	// Style
	fmt.Printf("Commit style (conventional/simple/emoji) %s: ", ui.Dim("[%s]", cfg.Style))
	style, _ := reader.ReadString('\n')
	if s := strings.TrimSpace(style); s != "" {
		cfg.Style = s
	}

	// Auto-push
	fmt.Printf("Auto-push after commit? (y/n) %s: ", ui.Dim("[y]"))
	autoPush, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(autoPush)) == "n" {
		cfg.AutoPush = false
	}

	// Include body
	fmt.Printf("Include commit body? (y/n) %s: ", ui.Dim("[y]"))
	includeBody, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(includeBody)) == "n" {
		cfg.IncludeBody = false
	}

	if err := Save(&cfg); err != nil {
		return nil, err
	}

	path, _ = ConfigPath()
	ui.Success("Config saved to %s", path)
	ui.Info("Continuing with your commit...")
	fmt.Println()

	return &cfg, nil
}

// EnsureConfig loads config or runs the wizard if no config file exists.
func EnsureConfig(flagOverrides map[string]string) (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	// Check if config file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Run setup wizard
		return RunSetupWizard()
	}

	return Load(flagOverrides)
}
