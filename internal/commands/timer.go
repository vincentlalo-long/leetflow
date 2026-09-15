package commands

import (
	"fmt"
	"strconv"
	"time"

	"leetcli/internal/config"
)

func Timer(args []string, cfg *config.Config, ui UI) error {
	ui.WriteOutput(MsgPlain, "--- Practice Stopwatch / Timer ---\n")

	minutes := 0
	if len(args) > 0 {
		m, err := strconv.Atoi(args[0])
		if err == nil && m > 0 {
			minutes = m
		}
	}

	if minutes == 0 {
		choices := []string{
			"1. Easy Problem (20 minutes)",
			"2. Medium Problem (30 minutes)",
			"3. Hard Problem (45 minutes)",
			"4. Custom Duration",
		}
		selected := ui.PromptSelect("Select practice target duration", choices)
		switch {
		case stringsHasPrefix(selected, "1."):
			minutes = 20
		case stringsHasPrefix(selected, "2."):
			minutes = 30
		case stringsHasPrefix(selected, "3."):
			minutes = 45
		case stringsHasPrefix(selected, "4."):
			input := ui.PromptText("Enter duration in minutes (e.g. 15)")
			m, err := strconv.Atoi(input)
			if err == nil && m > 0 {
				minutes = m
			}
		}
	}

	if minutes <= 0 {
		ui.WriteOutput(MsgError, "Invalid duration specified.")
		return fmt.Errorf("invalid duration specified")
	}

	ui.WriteOutput(MsgSuccess, "⏱️ Practice timer started for %d minutes!", minutes)
	ui.WriteOutput(MsgInfo, "Target end time: %s", time.Now().Add(time.Duration(minutes)*time.Minute).Format("15:04:05"))

	if isHeadless(ui) {
		return runHeadlessTimer(minutes, ui)
	}

	// Interactive (TUI) mode: fire the alarm in the background while the user
	// keeps working. The TUI event loop stays alive, so the goroutine survives.
	go fireTimerAlarm(minutes, ui)
	return nil
}

func isHeadless(ui UI) bool {
	h, ok := ui.(interface{ IsHeadless() bool })
	return ok && h.IsHeadless()
}

// runHeadlessTimer blocks until the duration elapses, reporting progress each
// minute. In CLI/headless mode a background goroutine would be killed the
// moment main returns, so the alarm is emitted synchronously instead.
func runHeadlessTimer(minutes int, ui UI) error {
	remaining := minutes
	for remaining > 0 {
		time.Sleep(time.Minute)
		remaining--
		if remaining > 0 {
			ui.WriteOutput(MsgInfo, "%d minute(s) remaining...", remaining)
		}
	}
	fireTimerAlarm(minutes, ui)
	return nil
}

func fireTimerAlarm(minutes int, ui UI) {
	ui.WriteOutput(MsgError, "\n⏰ ========================================")
	ui.WriteOutput(MsgError, "⏰ TIME IS UP! (%d minutes completed!)", minutes)
	ui.WriteOutput(MsgError, "⏰ ========================================\n")
}

func stringsHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
