package commands

import (
	"fmt"

	"leetcli/internal/api"
	"leetcli/internal/config"
)

func Profile(args []string, cfg *config.Config, ui UI) error {
	ui.WriteOutput(MsgPlain, "--- LeetCode User Profile ---\n")

	username := ""
	if len(args) > 0 {
		username = args[0]
	} else {
		username = ui.PromptText("Enter LeetCode username")
	}

	if username == "" {
		ui.WriteOutput(MsgError, "Username cannot be empty.")
		return fmt.Errorf("username cannot be empty")
	}

	ui.WriteOutput(MsgInfo, "Fetching profile for %s...", username)
	resp, err := api.GetUserProfile(username)
	if err != nil {
		ui.WriteOutput(MsgError, "User '%s' not found or an error occurred.", username)
		return fmt.Errorf("user '%s' not found or an error occurred: %w", username, err)
	}

	matched := resp.Data.MatchedUser
	if matched == nil {
		ui.WriteOutput(MsgError, "User '%s' not found.", username)
		return fmt.Errorf("user '%s' not found", username)
	}

	stats := matched.SubmitStats
	counts := map[string]int{}
	if stats != nil {
		for _, s := range stats.AcSubmissionNum {
			counts[s.Difficulty] = s.Count
		}
	}

	ranking := "N/A"
	reputation := 0
	if matched.Profile != nil {
		ranking = fmt.Sprintf("%.0f", matched.Profile.Ranking)
		reputation = matched.Profile.Reputation
	}

	ui.WriteOutput(MsgPlain, "\nProfile for %s:", username)
	ui.WriteOutput(MsgInfo, "Ranking:    %s", ranking)
	ui.WriteOutput(MsgInfo, "Reputation: %d", reputation)

	ui.WriteOutput(MsgPlain, "\nSolved Problems:")
	ui.WriteOutput(MsgInfo, "Total:  %d", counts["All"])
	ui.WriteOutput(MsgInfo, "Easy:   %d", counts["Easy"])
	ui.WriteOutput(MsgInfo, "Medium: %d", counts["Medium"])
	ui.WriteOutput(MsgInfo, "Hard:   %d", counts["Hard"])
	return nil
}
