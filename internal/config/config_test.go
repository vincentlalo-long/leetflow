package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()
	if cfg == nil {
		t.Fatal("Default() returned nil")
	}
	if cfg.DefaultLanguage != "cpp" {
		t.Errorf("DefaultLanguage = %q, want cpp", cfg.DefaultLanguage)
	}
	if len(cfg.DataStructures) == 0 {
		t.Errorf("DataStructures should not be empty")
	}
}

func TestAddRemoveDataStructure(t *testing.T) {
	cfg := Default()
	if !cfg.AddDataStructure("trie", "trie") {
		t.Errorf("AddDataStructure(trie) failed")
	}
	if cfg.GetDataStructures()["trie"] != "trie" {
		t.Errorf("GetDataStructures()[trie] = %q, want trie", cfg.GetDataStructures()["trie"])
	}
	// Adding again should return false
	if cfg.AddDataStructure("trie", "trie") {
		t.Errorf("AddDataStructure duplicate should return false")
	}

	if !cfg.RemoveDataStructure("trie") {
		t.Errorf("RemoveDataStructure(trie) failed")
	}
	if cfg.RemoveDataStructure("nonexistent") {
		t.Errorf("RemoveDataStructure(nonexistent) should return false")
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")

	cfg := Default()
	cfg.SetPath(cfgPath)
	cfg.LeetcodeSession = "test_session_123"
	cfg.LeetcodeCsrf = "test_csrf_456"

	if err := cfg.Save(); err != nil {
		t.Fatalf("cfg.Save() failed: %v", err)
	}

	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Fatalf("config file was not created at %s", cfgPath)
	}

	loaded, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load(cfgPath) failed: %v", err)
	}
	if loaded.LeetcodeSession != "test_session_123" {
		t.Errorf("LeetcodeSession = %q, want test_session_123", loaded.LeetcodeSession)
	}
	if loaded.LeetcodeCsrf != "test_csrf_456" {
		t.Errorf("LeetcodeCsrf = %q, want test_csrf_456", loaded.LeetcodeCsrf)
	}
}

func TestLoadOverlaysLocalConfig(t *testing.T) {
	tmpDir := t.TempDir()

	workspaceDir := filepath.Join(tmpDir, "workspace")
	os.MkdirAll(workspaceDir, 0755)

	baseCfg := `{"base_dir": "", "default_language": "cpp", "editor": "code"}`
	localMap := map[string]interface{}{
		"base_dir":         workspaceDir,
		"editor":           "vim",
		"leetcode_session": "local_session",
	}
	localData, _ := json.Marshal(localMap)

	os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte(baseCfg), 0644)
	os.WriteFile(filepath.Join(tmpDir, "config.local.json"), localData, 0644)

	cfg, err := Load(filepath.Join(tmpDir, "config.json"))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.BaseDir != workspaceDir {
		t.Errorf("BaseDir = %q, want %q (overlay should win)", cfg.BaseDir, workspaceDir)
	}
	if cfg.Editor != "vim" {
		t.Errorf("Editor = %q, want vim", cfg.Editor)
	}
	if cfg.DefaultLanguage != "cpp" {
		t.Errorf("DefaultLanguage = %q, want cpp (base value preserved)", cfg.DefaultLanguage)
	}
	if cfg.LeetcodeSession != "local_session" {
		t.Errorf("LeetcodeSession = %q, want local_session", cfg.LeetcodeSession)
	}
}

func TestEnvVarsOverrideCookies(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	os.WriteFile(cfgPath, []byte(`{}`), 0644)

	os.Setenv("LEETCODE_SESSION", "env_session")
	os.Setenv("LEETCODE_CSRF", "env_csrf")
	defer os.Unsetenv("LEETCODE_SESSION")
	defer os.Unsetenv("LEETCODE_CSRF")

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.GetLeetcodeSession() != "env_session" {
		t.Errorf("GetLeetcodeSession() = %q, want env_session", cfg.GetLeetcodeSession())
	}
	if cfg.GetLeetcodeCsrf() != "env_csrf" {
		t.Errorf("GetLeetcodeCsrf() = %q, want env_csrf", cfg.GetLeetcodeCsrf())
	}
}

func TestResolveBaseDir(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".git"), 0755)

	cfg := Default()
	cfg.SetPath(filepath.Join(tmpDir, "config.local.json"))
	cfg.BaseDir = ""
	cfg.ResolveBaseDir()

	if cfg.BaseDir != tmpDir {
		t.Errorf("ResolveBaseDir() = %q, want %q (repo root with .git)", cfg.BaseDir, tmpDir)
	}
}
