package report

import (
	"fmt"
	"strings"
	"time"
)

type Comparison struct {
	BeforeTitle string
	AfterTitle  string
	Changes     []CommandChange
}

type CommandChange struct {
	Command       string
	BeforeStatus  int
	AfterStatus   int
	DurationDelta time.Duration
	LineDelta     int
	State         string
}

func Compare(before, after Report) Comparison {
	comparison := Comparison{BeforeTitle: before.Title, AfterTitle: after.Title}
	beforeByCommand := map[string]CommandResult{}
	for _, command := range before.Commands {
		beforeByCommand[command.Command] = command
	}
	seen := map[string]bool{}
	for _, current := range after.Commands {
		previous, found := beforeByCommand[current.Command]
		change := CommandChange{Command: current.Command, AfterStatus: current.ExitCode, BeforeStatus: -1, State: "added", LineDelta: lineCount(current.Output)}
		if found {
			change.BeforeStatus = previous.ExitCode
			change.DurationDelta = current.Duration - previous.Duration
			change.LineDelta = lineCount(current.Output) - lineCount(previous.Output)
			change.State = "unchanged"
			if previous.ExitCode == 0 && current.ExitCode != 0 {
				change.State = "regressed"
			}
			if previous.ExitCode != 0 && current.ExitCode == 0 {
				change.State = "fixed"
			}
			if change.State == "unchanged" && (change.LineDelta != 0 || change.DurationDelta != 0) {
				change.State = "changed"
			}
		}
		comparison.Changes = append(comparison.Changes, change)
		seen[current.Command] = true
	}
	for _, previous := range before.Commands {
		if !seen[previous.Command] {
			comparison.Changes = append(comparison.Changes, CommandChange{Command: previous.Command, BeforeStatus: previous.ExitCode, AfterStatus: -1, State: "removed", LineDelta: -lineCount(previous.Output)})
		}
	}
	return comparison
}

func (c Comparison) Markdown() string {
	var output strings.Builder
	fmt.Fprintf(&output, "# Proofshot comparison\n\n**Before:** %s  \n**After:** %s\n\n", c.BeforeTitle, c.AfterTitle)
	output.WriteString("| State | Command | Exit | Time Δ | Lines Δ |\n|---|---|---:|---:|---:|\n")
	for _, change := range c.Changes {
		exit := fmt.Sprintf("%d → %d", change.BeforeStatus, change.AfterStatus)
		fmt.Fprintf(&output, "| %s | `%s` | %s | %s | %+d |\n", change.State, strings.ReplaceAll(change.Command, "|", "\\|"), exit, signedDuration(change.DurationDelta), change.LineDelta)
	}
	return output.String()
}

func lineCount(value string) int {
	if value == "" {
		return 0
	}
	return strings.Count(value, "\n") + 1
}

func signedDuration(value time.Duration) string {
	if value > 0 {
		return "+" + value.Round(time.Millisecond).String()
	}
	return value.Round(time.Millisecond).String()
}
