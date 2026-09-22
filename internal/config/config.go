package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const (
	localConfigName = "config.local.json"
	envSession      = "LEETCODE_SESSION"
	envCsrf         = "LEETCODE_CSRF"
)

type LangEntry struct {
	Label string `json:"label"`
	Ext   string `json:"ext"`
}

type Config struct {
	BaseDir         string               `json:"base_dir"`
	Languages       map[string]LangEntry `json:"languages"`
	DefaultLanguage string               `json:"default_language"`
	DataStructures  map[string]string    `json:"data_structures"`
	Theme           string               `json:"theme"`
	Editor          string               `json:"editor"`
	Terminal        string               `json:"terminal,omitempty"`
	OpenMode        string               `json:"open_mode,omitempty"`
	LeetcodeSession string               `json:"leetcode_session,omitempty"`
	LeetcodeCsrf    string               `json:"leetcode_csrf,omitempty"`

	path           string
	hasLocalConfig bool
}

// Load reads the base config (config.json) and overlays the local, git-ignored
// config.local.json if present. Secrets (session / csrf) are kept out of the
// committed config.json and can also be provided via env vars.
func Load(path string) (*Config, error) {
	dir := ""
	if path != "" {
		dir = filepath.Dir(path)
	} else {
		for _, p := range []string{
			"config.json",
			filepath.Join("..", "config.json"),
			filepath.Join("..", "..", "config.json"),
		} {
			if _, err := os.Stat(p); err == nil {
				dir = filepath.Dir(p)
				break
			}
		}
		if dir == "" {
			if uDir, err := os.UserConfigDir(); err == nil {
				candidate := filepath.Join(uDir, "leet", "config.json")
				if _, err := os.Stat(candidate); err == nil {
					dir = filepath.Dir(candidate)
				}
			}
		}
	}

	cfg := Default()
	if dir != "" {
		if data, err := os.ReadFile(filepath.Join(dir, "config.json")); err == nil {
			json.Unmarshal(data, cfg)
		}
	}

	// Overlay local (git-ignored) settings, e.g. machine-specific base_dir.
	// Save() always writes to this local file so secrets never reach git.
	localPath := filepath.Join(dir, localConfigName)
	if _, err := os.Stat(localPath); err == nil {
		cfg.hasLocalConfig = true
		if data, err := os.ReadFile(localPath); err == nil {
			var local map[string]interface{}
			if json.Unmarshal(data, &local) == nil {
				applyOverlay(cfg, local)
			}
		}
	}

	if cfg.path == "" {
		if dir != "" {
			cfg.path = localPath
		} else {
			cfg.path = "config.json"
		}
	}
	cfg.path, _ = filepath.Abs(cfg.path)

	cfg.ResolveBaseDir()
	return cfg, nil
}

func applyOverlay(cfg *Config, overlay map[string]interface{}) {
	if v, ok := overlay["base_dir"].(string); ok && v != "" {
		cfg.BaseDir = v
	}
	if v, ok := overlay["default_language"].(string); ok && v != "" {
		cfg.DefaultLanguage = v
	}
	if v, ok := overlay["editor"].(string); ok && v != "" {
		cfg.Editor = v
	}
	if v, ok := overlay["terminal"].(string); ok && v != "" {
		cfg.Terminal = v
	}
	if v, ok := overlay["open_mode"].(string); ok && v != "" {
		cfg.OpenMode = v
	}
	if v, ok := overlay["theme"].(string); ok && v != "" {
		cfg.Theme = v
	}
	if v, ok := overlay["leetcode_session"].(string); ok && v != "" {
		cfg.LeetcodeSession = v
	}
	if v, ok := overlay["leetcode_csrf"].(string); ok && v != "" {
		cfg.LeetcodeCsrf = v
	}
	if v, ok := overlay["languages"].(map[string]interface{}); ok && len(v) > 0 {
		langs := make(map[string]LangEntry)
		for k, raw := range v {
			if m, ok := raw.(map[string]interface{}); ok {
				label, _ := m["label"].(string)
				ext, _ := m["ext"].(string)
				if ext != "" {
					langs[k] = LangEntry{Label: label, Ext: ext}
				}
			}
		}
		if len(langs) > 0 {
			cfg.Languages = langs
		}
	}
	if v, ok := overlay["data_structures"].(map[string]interface{}); ok && len(v) > 0 {
		ds := make(map[string]string)
		for k, raw := range v {
			if s, ok := raw.(string); ok && s != "" {
				ds[k] = s
			}
		}
		if len(ds) > 0 {
			cfg.DataStructures = ds
		}
	}
}

// ExpandHome replaces leading ~/ with the user's home directory path.
func ExpandHome(path string) string {
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				return home
			}
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// ResolveBaseDir makes base_dir machine-independent: if it is empty or the
// directory does not exist, walk up from the config file to find the repo root
// (the directory containing .git).
func (c *Config) ResolveBaseDir() {
	if c.BaseDir != "" {
		c.BaseDir = ExpandHome(c.BaseDir)
		if info, err := os.Stat(c.BaseDir); err == nil && info.IsDir() {
			return
		}
	}
	if c.BaseDir == "" {
		c.BaseDir = DetectBaseDir(c.path)
	}
}

func DetectBaseDir(configPath string) string {
	start := configPath
	if start == "" {
		if cwd, err := os.Getwd(); err == nil {
			start = filepath.Join(cwd, "config.json")
		}
	}
	dir := filepath.Dir(start)
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Dir(start)
}

func Default() *Config {
	return &Config{
		BaseDir:         "",
		DefaultLanguage: "cpp",
		Theme:           "default",
		Editor:          "code",
		Languages: map[string]LangEntry{
			"cpp":        {Label: "C++", Ext: "cpp"},
			"c":          {Label: "C", Ext: "c"},
			"python":     {Label: "Python", Ext: "py"},
			"go":         {Label: "Go", Ext: "go"},
			"java":       {Label: "Java", Ext: "java"},
			"rust":       {Label: "Rust", Ext: "rs"},
			"javascript": {Label: "JavaScript", Ext: "js"},
			"typescript": {Label: "TypeScript", Ext: "ts"},
			"csharp":     {Label: "C#", Ext: "cs"},
		},
		DataStructures: map[string]string{
			"array":        "array",
			"string":       "string",
			"linkedlist":   "linked_list",
			"stack":        "stack",
			"queue":        "queue",
			"graph":        "graph",
			"tree":         "tree",
			"heap":         "heap",
			"hash":         "hash",
			"dp":           "dp",
			"binary":       "binary",
			"two-pointer":  "two_pointers",
			"sliding":      "sliding_window",
			"backtracking": "backtracking",
			"greedy":       "greedy",
			"math":         "math",
		},
	}
}

func (c *Config) GetPath() string {
	if c.path == "" {
		return "config.json"
	}
	return c.path
}

func (c *Config) SetPath(p string) {
	if p == "" {
		p = "config.json"
	}
	abs, err := filepath.Abs(p)
	if err == nil {
		c.path = abs
	} else {
		c.path = p
	}
}

// Save writes to the local, git-ignored config.local.json so secrets never
// leak into the committed config.json.
func (c *Config) Save() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	targetPath := c.GetPath()
	if dir := filepath.Dir(targetPath); dir != "" {
		_ = os.MkdirAll(dir, 0700)
	}
	return os.WriteFile(targetPath, data, 0600)
}

func (c *Config) GetDataStructures() map[string]string {
	if c.DataStructures == nil {
		return map[string]string{}
	}
	return c.DataStructures
}

func (c *Config) AddDataStructure(name, folder string) bool {
	if c.DataStructures == nil {
		c.DataStructures = make(map[string]string)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	if _, ok := c.DataStructures[name]; ok {
		return false
	}
	c.DataStructures[name] = folder
	return true
}

func (c *Config) RemoveDataStructure(name string) bool {
	if _, ok := c.DataStructures[name]; !ok {
		return false
	}
	delete(c.DataStructures, name)
	return true
}

func (c *Config) SetTheme(theme string) {
	c.Theme = theme
}

func (c *Config) GetTheme() string {
	if c.Theme == "" {
		return "default"
	}
	return c.Theme
}

func (c *Config) GetEditor() string {
	if c.Editor == "" {
		return "code"
	}
	return c.Editor
}

func (c *Config) SetEditor(editor string) {
	c.Editor = editor
}

// GetLeetcodeSession returns the cookie from env var first, falling back to
// the local config value. Keeps secrets out of git.
func (c *Config) GetLeetcodeSession() string {
	if v := os.Getenv(envSession); v != "" {
		return v
	}
	return c.LeetcodeSession
}

// GetLeetcodeCsrf returns the csrf token from env var first, falling back to
// the local config value.
func (c *Config) GetLeetcodeCsrf() string {
	if v := os.Getenv(envCsrf); v != "" {
		return v
	}
	return c.LeetcodeCsrf
}

func (c *Config) HasLocalConfig() bool {
	return c.hasLocalConfig
}

func (c *Config) NeedsSetup() bool {
	return !c.hasLocalConfig || c.BaseDir == ""
}

func (c *Config) GetTerminal() string {
	if c.Terminal == "" {
		return "kitty"
	}
	return c.Terminal
}

func (c *Config) SetTerminal(term string) {
	c.Terminal = term
}

func (c *Config) GetOpenMode() string {
	if c.OpenMode == "" {
		return "auto"
	}
	return c.OpenMode
}

func (c *Config) SetOpenMode(mode string) {
	c.OpenMode = mode
}
