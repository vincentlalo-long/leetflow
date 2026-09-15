package commands

import (
	"reflect"
	"testing"
)

func TestParseFlags(t *testing.T) {
	pos, flags := parseFlags([]string{"1", "--lang", "go", "--ds", "array", "--name", "Two Sum", "--unsolved"})

	wantPos := []string{"1"}
	if !reflect.DeepEqual(pos, wantPos) {
		t.Errorf("pos = %v, want %v", pos, wantPos)
	}
	wantFlags := map[string]string{"lang": "go", "ds": "array", "name": "Two Sum", "unsolved": "true"}
	if !reflect.DeepEqual(flags, wantFlags) {
		t.Errorf("flags = %v, want %v", flags, wantFlags)
	}
}

func TestParseFlagsEq(t *testing.T) {
	pos, flags := parseFlags([]string{"--lang=go", "2"})
	if len(pos) != 1 || pos[0] != "2" {
		t.Errorf("pos = %v, want [2]", pos)
	}
	if flags["lang"] != "go" {
		t.Errorf("flags[lang] = %q, want go", flags["lang"])
	}
}

func TestHasFlag(t *testing.T) {
	if !hasFlag(map[string]string{"yes": "true"}, "yes") {
		t.Errorf("hasFlag(true) should be true")
	}
	if hasFlag(map[string]string{"yes": "false"}, "yes") {
		t.Errorf("hasFlag(false) should be false")
	}
	if hasFlag(map[string]string{}, "nope") {
		t.Errorf("hasFlag(missing) should be false")
	}
}

func TestIsDifficulty(t *testing.T) {
	if !isDifficulty("Easy") || !isDifficulty("MEDIUM") || !isDifficulty("hard") {
		t.Errorf("difficulty detection failed")
	}
	if isDifficulty("array") {
		t.Errorf("array should not be a difficulty")
	}
}
