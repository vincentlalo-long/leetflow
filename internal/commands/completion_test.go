package commands

import (
	"fmt"
	"strings"
	"testing"

	"leetcli/internal/config"
)

// fakeUI is a minimal UI for testing that records output.
type fakeUI struct {
	written strings.Builder
}

func (f *fakeUI) PromptText(label string) string                          { return "" }
func (f *fakeUI) PromptSelect(label string, choices []string) string      { return "" }
func (f *fakeUI) PromptConfirm(label string) bool                         { return false }
func (f *fakeUI) WriteOutput(kind MsgKind, format string, args ...interface{}) {
	f.written.WriteString(fmt.Sprintf(format, args...))
}
func (f *fakeUI) WriteString(s string) { f.written.WriteString(s) }
func (f *fakeUI) Writef(format string, args ...interface{}) {
	f.written.WriteString(fmt.Sprintf(format, args...))
}
func (f *fakeUI) Output() string { return f.written.String() }

func TestCompletionBash(t *testing.T) {
	ui := &fakeUI{}
	cfg := config.Default()
	err := CompletionCommand([]string{"bash"}, cfg, ui)
	if err != nil {
		t.Fatal(err)
	}
	out := ui.Output()
	if !strings.Contains(out, "_leet_completions") {
		t.Errorf("bash completion missing function:\n%s", out)
	}
	if !strings.Contains(out, "complete -F") {
		t.Errorf("bash completion missing complete directive:\n%s", out)
	}
	for _, cmd := range []string{"add", "list", "test", "doctor", "verify", "completion"} {
		if !strings.Contains(out, cmd) {
			t.Errorf("bash completion missing command %q:\n%s", cmd, out)
		}
	}
}

func TestCompletionZsh(t *testing.T) {
	ui := &fakeUI{}
	cfg := config.Default()
	err := CompletionCommand([]string{"zsh"}, cfg, ui)
	if err != nil {
		t.Fatal(err)
	}
	out := ui.Output()
	if !strings.Contains(out, "#compdef leet") {
		t.Errorf("zsh completion missing compdef:\n%s", out)
	}
	if !strings.Contains(out, "_describe") {
		t.Errorf("zsh completion missing _describe:\n%s", out)
	}
}

func TestCompletionFish(t *testing.T) {
	ui := &fakeUI{}
	cfg := config.Default()
	err := CompletionCommand([]string{"fish"}, cfg, ui)
	if err != nil {
		t.Fatal(err)
	}
	out := ui.Output()
	if !strings.Contains(out, "complete -c leet") {
		t.Errorf("fish completion missing complete directive:\n%s", out)
	}
	for _, cmd := range []string{"add", "list", "test"} {
		if !strings.Contains(out, "'"+cmd+"'") {
			t.Errorf("fish completion missing command %q:\n%s", cmd, out)
		}
	}
}

func TestCompletionUnknownShell(t *testing.T) {
	ui := &fakeUI{}
	cfg := config.Default()
	err := CompletionCommand([]string{"powershell"}, cfg, ui)
	if err == nil {
		t.Error("expected error for unknown shell")
	}
}

func TestCompletionDefaultIsBash(t *testing.T) {
	ui := &fakeUI{}
	cfg := config.Default()
	err := CompletionCommand([]string{}, cfg, ui)
	if err != nil {
		t.Fatal(err)
	}
	out := ui.Output()
	if !strings.Contains(out, "_leet_completions") {
		t.Errorf("default should be bash, got:\n%s", out)
	}
}
