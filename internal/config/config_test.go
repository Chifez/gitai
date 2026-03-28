package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyFlagOverrides(t *testing.T) {
	cfg := Defaults()
	flags := map[string]string{
		"model":      "gpt-force",
		"max_length": "100",
		"style":      "emoji",
	}
	applyFlagOverrides(&cfg, flags)

	if cfg.Model != "gpt-force" {
		t.Errorf("expected gpt-force, got %s", cfg.Model)
	}
	if cfg.MaxLength != 100 {
		t.Errorf("expected 100, got %d", cfg.MaxLength)
	}
	if cfg.Style != "emoji" {
		t.Errorf("expected emoji, got %s", cfg.Style)
	}
}

func TestApplyFlagOverrides_AllFields(t *testing.T) {
	cfg := Defaults()
	flags := map[string]string{
		"provider": "anthropic",
		"api_key":  "sk-test",
		"model":    "claude-3",
		"style":    "simple",
		"lang":     "french",
	}
	applyFlagOverrides(&cfg, flags)

	if cfg.Provider != "anthropic" {
		t.Errorf("expected anthropic, got %s", cfg.Provider)
	}
	if cfg.APIKey != "sk-test" {
		t.Errorf("expected sk-test, got %s", cfg.APIKey)
	}
	if cfg.Lang != "french" {
		t.Errorf("expected french, got %s", cfg.Lang)
	}
}

func TestApplyFlagOverrides_Nil(t *testing.T) {
	cfg := Defaults()
	applyFlagOverrides(&cfg, nil)
	// Should not panic and cfg should remain defaults
	if cfg.Model != "gpt-4o-mini" {
		t.Errorf("expected default model, got %s", cfg.Model)
	}
}

func TestApplyFlagOverrides_EmptyValues(t *testing.T) {
	cfg := Defaults()
	flags := map[string]string{
		"model": "",
		"style": "",
	}
	applyFlagOverrides(&cfg, flags)
	// Empty values should not override
	if cfg.Model != "gpt-4o-mini" {
		t.Errorf("expected default model, got %s", cfg.Model)
	}
}

func TestApplyFlagOverrides_InvalidMaxLength(t *testing.T) {
	cfg := Defaults()
	flags := map[string]string{
		"max_length": "not-a-number",
	}
	applyFlagOverrides(&cfg, flags)
	if cfg.MaxLength != 72 {
		t.Errorf("expected default 72, got %d", cfg.MaxLength)
	}
}

func TestApplyEnvOverrides(t *testing.T) {
	t.Setenv("GITAI_AUTO_PUSH", "false")
	t.Setenv("OPENAI_API_KEY", "env-key")

	cfg := Defaults()
	applyEnvOverrides(&cfg)

	if cfg.AutoPush != false {
		t.Errorf("expected auto_push false, got %v", cfg.AutoPush)
	}
	if cfg.APIKey != "env-key" {
		t.Errorf("expected env-key, got %s", cfg.APIKey)
	}
}

func TestApplyEnvOverrides_AllVars(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-env")
	t.Setenv("GITAI_MODEL", "gpt-4o")
	t.Setenv("GITAI_PROVIDER", "mock")
	t.Setenv("GITAI_STYLE", "emoji")
	t.Setenv("GITAI_AUTO_PUSH", "true")
	t.Setenv("GITAI_LANG", "spanish")

	cfg := Defaults()
	applyEnvOverrides(&cfg)

	if cfg.APIKey != "sk-env" {
		t.Errorf("expected sk-env, got %s", cfg.APIKey)
	}
	if cfg.Model != "gpt-4o" {
		t.Errorf("expected gpt-4o, got %s", cfg.Model)
	}
	if cfg.Provider != "mock" {
		t.Errorf("expected mock, got %s", cfg.Provider)
	}
	if cfg.Style != "emoji" {
		t.Errorf("expected emoji, got %s", cfg.Style)
	}
	if cfg.Lang != "spanish" {
		t.Errorf("expected spanish, got %s", cfg.Lang)
	}
}

func TestSaveAndLoad(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	cfg := Defaults()
	cfg.APIKey = "test-sk"

	if err := Save(&cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(nil)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.APIKey != "test-sk" {
		t.Errorf("expected test-sk, got %s", loaded.APIKey)
	}
	
	val, src, err := GetValue("api_key")
	if err != nil {
		t.Fatalf("GetValue failed: %v", err)
	}
	if src != "file" {
		t.Errorf("expected source 'file', got %s", src)
	}
	if val != "test..." {
		t.Errorf("expected masked key, got %s", val)
	}
}

func TestGetValueShortAPIKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	cfg := Defaults()
	cfg.APIKey = "ab"
	if err := Save(&cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	val, src, err := GetValue("api_key")
	if err != nil {
		t.Fatalf("GetValue failed: %v", err)
	}
	if src != "file" {
		t.Errorf("expected source 'file', got %s", src)
	}
	if val != "***" {
		t.Errorf("expected *** for short key, got %s", val)
	}
}

func TestGetValue_AllKeys(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	cfg := Defaults()
	cfg.APIKey = "test-key-12345"
	if err := Save(&cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	keys := []string{
		"provider", "api_key", "model", "style", "auto_push",
		"max_length", "lang", "include_body", "default_remote_name", "auto_set_upstream",
	}

	for _, key := range keys {
		val, _, err := GetValue(key)
		if err != nil {
			t.Errorf("GetValue(%s) failed: %v", key, err)
		}
		if val == "" {
			t.Errorf("GetValue(%s) returned empty string", key)
		}
	}
}

func TestGetValue_UnknownKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	cfg := Defaults()
	if err := Save(&cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	_, _, err := GetValue("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
}

func TestSetValue_AllKeys(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	cfg := Defaults()
	if err := Save(&cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	cases := []struct {
		key   string
		value string
	}{
		{"provider", "mock"},
		{"api_key", "sk-new"},
		{"model", "gpt-4o"},
		{"style", "emoji"},
		{"auto_push", "false"},
		{"max_length", "100"},
		{"lang", "french"},
		{"include_body", "false"},
		{"default_remote_name", "upstream"},
		{"auto_set_upstream", "false"},
	}

	for _, tc := range cases {
		if err := SetValue(tc.key, tc.value); err != nil {
			t.Errorf("SetValue(%s, %s) failed: %v", tc.key, tc.value, err)
		}
	}

	loaded, _ := Load(nil)
	if loaded.Provider != "mock" {
		t.Errorf("expected mock, got %s", loaded.Provider)
	}
	if loaded.Style != "emoji" {
		t.Errorf("expected emoji, got %s", loaded.Style)
	}
	if loaded.AutoPush != false {
		t.Errorf("expected false, got %v", loaded.AutoPush)
	}
	if loaded.MaxLength != 100 {
		t.Errorf("expected 100, got %d", loaded.MaxLength)
	}
	if loaded.IncludeBody != false {
		t.Errorf("expected false, got %v", loaded.IncludeBody)
	}
	if loaded.DefaultRemoteName != "upstream" {
		t.Errorf("expected upstream, got %s", loaded.DefaultRemoteName)
	}
	if loaded.AutoSetUpstream != false {
		t.Errorf("expected false, got %v", loaded.AutoSetUpstream)
	}
}

func TestSetValue_InvalidMaxLength(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	cfg := Defaults()
	if err := Save(&cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	err := SetValue("max_length", "not-a-number")
	if err == nil {
		t.Fatal("expected error for invalid max_length")
	}
}

func TestSetValue_UnknownKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	cfg := Defaults()
	if err := Save(&cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	err := SetValue("nonexistent", "value")
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
}

func TestListAll(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	cfg := Defaults()
	cfg.APIKey = "sk-list-test"
	if err := Save(&cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	entries, err := ListAll()
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}
	if len(entries) != 10 {
		t.Errorf("expected 10 entries, got %d", len(entries))
	}

	// Check that api_key is masked
	for _, e := range entries {
		if e.Key == "api_key" {
			if e.Value != "sk-l..." {
				t.Errorf("expected masked api_key, got %s", e.Value)
			}
		}
	}
}

func TestDefaults(t *testing.T) {
	cfg := Defaults()
	if cfg.Provider != "openai" {
		t.Errorf("expected openai, got %s", cfg.Provider)
	}
	if cfg.Model != "gpt-4o-mini" {
		t.Errorf("expected gpt-4o-mini, got %s", cfg.Model)
	}
	if cfg.Style != "conventional" {
		t.Errorf("expected conventional, got %s", cfg.Style)
	}
	if !cfg.AutoPush {
		t.Error("expected AutoPush true")
	}
	if cfg.MaxLength != 72 {
		t.Errorf("expected 72, got %d", cfg.MaxLength)
	}
	if cfg.Lang != "english" {
		t.Errorf("expected english, got %s", cfg.Lang)
	}
	if !cfg.IncludeBody {
		t.Error("expected IncludeBody true")
	}
	if cfg.DefaultRemoteName != "origin" {
		t.Errorf("expected origin, got %s", cfg.DefaultRemoteName)
	}
	if !cfg.AutoSetUpstream {
		t.Error("expected AutoSetUpstream true")
	}
}

func TestConfigPath(t *testing.T) {
	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath failed: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path")
	}
	if filepath.Base(path) != "config.yaml" {
		t.Errorf("expected config.yaml, got %s", filepath.Base(path))
	}
}

func TestResolveSource_Default(t *testing.T) {
	// Clear env vars to test default source
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("GITAI_MODEL", "")

	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	// Don't save config file — should get "default"

	src := resolveSource("model")
	if src != "default" {
		t.Errorf("expected default, got %s", src)
	}
}

func TestResolveSource_Env(t *testing.T) {
	t.Setenv("GITAI_MODEL", "gpt-4o")
	src := resolveSource("model")
	if src != "env" {
		t.Errorf("expected env, got %s", src)
	}
}

func TestResolveSource_File(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	t.Setenv("GITAI_MODEL", "")

	cfg := Defaults()
	if err := Save(&cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	src := resolveSource("model")
	if src != "file" {
		t.Errorf("expected file, got %s", src)
	}
}

func TestBuildProvider_OpenAI(t *testing.T) {
	cfg := Defaults()
	cfg.APIKey = "sk-test"
	p, err := cfg.BuildProvider()
	if err != nil {
		t.Fatalf("BuildProvider failed: %v", err)
	}
	if p.Name() != "openai" {
		t.Errorf("expected openai, got %s", p.Name())
	}
}

func TestBuildProvider_Mock(t *testing.T) {
	cfg := Defaults()
	cfg.Provider = "mock"
	p, err := cfg.BuildProvider()
	if err != nil {
		t.Fatalf("BuildProvider failed: %v", err)
	}
	if p.Name() != "mock" {
		t.Errorf("expected mock, got %s", p.Name())
	}
}

func TestBuildProvider_NoAPIKey(t *testing.T) {
	cfg := Defaults()
	cfg.APIKey = ""
	_, err := cfg.BuildProvider()
	if err == nil {
		t.Fatal("expected error when API key is missing")
	}
}

func TestBuildProvider_Unknown(t *testing.T) {
	cfg := Defaults()
	cfg.Provider = "invalid"
	_, err := cfg.BuildProvider()
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestEnsureConfig_ExistingFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	cfg := Defaults()
	cfg.Style = "emoji"
	if err := Save(&cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := EnsureConfig(nil)
	if err != nil {
		t.Fatalf("EnsureConfig failed: %v", err)
	}
	if loaded.Style != "emoji" {
		t.Errorf("expected emoji, got %s", loaded.Style)
	}
}

func TestLoad_WithFlagOverrides(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	cfg := Defaults()
	if err := Save(&cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	flags := map[string]string{
		"model": "gpt-4o",
		"style": "simple",
	}
	loaded, err := Load(flags)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.Model != "gpt-4o" {
		t.Errorf("expected gpt-4o, got %s", loaded.Model)
	}
	if loaded.Style != "simple" {
		t.Errorf("expected simple, got %s", loaded.Style)
	}
}

func TestLoad_NoConfigFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	// Don't create config file — Load should still work with defaults
	loaded, err := Load(nil)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.Model != "gpt-4o-mini" {
		t.Errorf("expected default model, got %s", loaded.Model)
	}
}

func TestSave_CreatesDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	cfg := Defaults()
	if err := Save(&cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify directory was created
	configDir := filepath.Join(home, ".gitai")
	info, err := os.Stat(configDir)
	if err != nil {
		t.Fatalf("config dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory, got file")
	}
}
