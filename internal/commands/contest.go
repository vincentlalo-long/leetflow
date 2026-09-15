package commands

import (
	"fmt"
	"time"

	"leetcli/internal/api"
	"leetcli/internal/config"
)

func Contest(args []string, cfg *config.Config, ui UI) error {
	ui.WriteOutput(MsgPlain, "--- Upcoming LeetCode Contests ---\n")

	ui.WriteOutput(MsgInfo, "Fetching contest information...")
	contests, err := api.GetUpcomingContests()
	if err != nil {
		ui.WriteOutput(MsgError, "Could not fetch contest information.")
		return fmt.Errorf("could not fetch contest information: %w", err)
	}

	if len(contests) == 0 {
		ui.WriteOutput(MsgInfo, "No upcoming contests found.")
		return nil
	}

	ui.WriteOutput(MsgPlain, "Upcoming Contests:")
	for _, c := range contests {
		startTime := time.Unix(c.StartTime, 0)
		durationHours := float64(c.Duration) / 3600
		link := "https://leetcode.com/contest/" + c.TitleSlug + "/"

		ui.WriteOutput(MsgPlain, "  %s", c.Title)
		ui.WriteOutput(MsgInfo, "    Start: %s", startTime.Format("2006-01-02 15:04:05"))
		ui.WriteOutput(MsgInfo, "    Duration: %.0f hours", durationHours)
		ui.WriteOutput(MsgInfo, "    Link: %s", link)
	}
	return nil
}
