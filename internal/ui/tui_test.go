package ui

import (
	"strings"
	"testing"

	"leetcli/internal/config"
)

func TestRenderBanner(t *testing.T) {
	cfg := config.Default()
	banner := RenderBanner(80, cfg)
	if !strings.Contains(banner, "LEETFLOW") {
		t.Errorf("expected LEETFLOW in banner, got: %s", banner)
	}
	if !strings.Contains(banner, "help") {
		t.Errorf("expected help hint in banner, got: %s", banner)
	}
}

func TestRenderStatusBar(t *testing.T) {
	cfg := config.Default()
	cfg.BaseDir = "/tmp/test-leetcode"
	cfg.DefaultLanguage = "cpp"
	cfg.Editor = "nvim"

	bar := RenderStatusBar(80, cfg)
	if !strings.Contains(bar, "cpp") {
		t.Errorf("expected 'cpp' in status bar, got: %s", bar)
	}
	if !strings.Contains(bar, "nvim") {
		t.Errorf("expected 'nvim' in status bar, got: %s", bar)
	}
	if strings.Contains(bar, "IDEAL") {
		t.Errorf("status bar should not contain legacy 'IDEAL', got: %s", bar)
	}
	if strings.Contains(bar, "no sandbox") {
		t.Errorf("status bar should not contain legacy 'no sandbox', got: %s", bar)
	}
}

func TestTUIViewNoRedundantDividers(t *testing.T) {
	cfg := config.Default()
	cfg.BaseDir = "/tmp/test-leetcode"
	m := New(cfg)
	m.ready = true
	m.termWidth = 80
	m.termHeight = 24

	view := m.View()
	if strings.Contains(view, "IDEAL") {
		t.Errorf("view should not contain legacy IDEAL banner: %s", view)
	}

	// Count number of horizontal divider lines (strings of ─)
	dividerCount := strings.Count(view, strings.Repeat("─", 80))
	// Exactly 2 dividers: 1 under banner, 1 above status bar
	if dividerCount != 2 {
		t.Errorf("expected exactly 2 dividers in TUI, found %d: %s", dividerCount, view)
	}
}
