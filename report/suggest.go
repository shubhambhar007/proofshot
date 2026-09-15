package report

import (
	"fmt"
	"sort"
	"strings"
)

// Suggestion explains an evidence-based recommendation, not a causal claim.
type Suggestion struct {
	Number int
	Reason string
}

func SuggestCommands(source Report) []Suggestion {
	if len(source.Commands) == 0 {
		return nil
	}
	reasons := map[int]string{}
	firstFailure := -1
	for index, command := range source.Commands {
		if command.ExitCode != 0 || command.TimedOut {
			if firstFailure < 0 {
				firstFailure = index
			}
			reasons[index+1] = fmt.Sprintf("Failed with exit %d", command.ExitCode)
			if command.TimedOut {
				reasons[index+1] = "Timed out"
			}
		}
	}
	if firstFailure < 0 {
		// A clean session still needs a concise handoff: show the first and final checks.
		reasons[1] = "First recorded step"
		reasons[len(source.Commands)] = "Final recorded step"
	} else {
		if firstFailure > 0 {
			previous := firstFailure
			if _, exists := reasons[previous]; !exists {
				reasons[previous] = "Immediately before the first failure (sequence only; not proof of cause)"
			}
		}
		for index := firstFailure + 1; index < len(source.Commands); index++ {
			command := source.Commands[index]
			if command.ExitCode == 0 && !command.TimedOut {
				if _, exists := reasons[index+1]; !exists {
					reasons[index+1] = "First successful step after the failure"
				}
				break
			}
		}
	}
	numbers := make([]int, 0, len(reasons))
	for number := range reasons {
		numbers = append(numbers, number)
	}
	sort.Ints(numbers)
	suggestions := make([]Suggestion, 0, len(numbers))
	for _, number := range numbers {
		suggestions = append(suggestions, Suggestion{Number: number, Reason: reasons[number]})
	}
	return suggestions
}

func SummarizeHandoff(commands []CommandResult) string {
	if len(commands) == 0 {
		return "No command evidence selected."
	}
	failed := 0
	firstFailure := -1
	for index, command := range commands {
		if command.ExitCode != 0 || command.TimedOut {
			failed++
			if firstFailure < 0 {
				firstFailure = index
			}
		}
	}
	if firstFailure < 0 {
		return fmt.Sprintf("All %d selected commands completed successfully; no failure was captured in this handoff.", len(commands))
	}
	command := commands[firstFailure]
	result := fmt.Sprintf("%d of %d selected commands failed. The first captured failure was `%s` (exit %d).", failed, len(commands), command.Command, command.ExitCode)
	if len(command.Findings) > 0 {
		result += " Its output includes: " + strings.TrimSpace(command.Findings[0].Summary) + "."
	}
	if firstFailure > 0 {
		result += " The previous selected command is shown for sequence context, not as a confirmed cause."
	}
	return result
}
