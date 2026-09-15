package report

import (
	"fmt"
	"sort"
	"strings"
)

type BaselineDelta struct {
	State      string
	BeforeExit int
	AfterExit  int
	BeforeLine string
	AfterLine  string
}

// MatchBaseline uses command text and occurrence order, so repeated commands
// are paired with the corresponding occurrence rather than silently collapsed.
func MatchBaseline(before, after Report) map[int]BaselineDelta {
	byCommand := map[string][]CommandResult{}
	for _, command := range before.Commands {
		byCommand[command.Command] = append(byCommand[command.Command], command)
	}
	used := map[string]int{}
	deltas := map[int]BaselineDelta{}
	for index, current := range after.Commands {
		occurrence := used[current.Command]
		used[current.Command]++
		previousList := byCommand[current.Command]
		if occurrence >= len(previousList) {
			deltas[index+1] = BaselineDelta{State: "not in baseline", BeforeExit: -1, AfterExit: current.ExitCode}
			continue
		}
		previous := previousList[occurrence]
		delta := BaselineDelta{BeforeExit: previous.ExitCode, AfterExit: current.ExitCode, State: "same result"}
		switch {
		case previous.ExitCode == 0 && current.ExitCode != 0:
			delta.State = "regressed"
		case previous.ExitCode != 0 && current.ExitCode == 0:
			delta.State = "fixed"
		case previous.ExitCode != current.ExitCode:
			delta.State = "exit changed"
		case StripANSI(previous.Output) != StripANSI(current.Output):
			delta.State = "output changed"
		}
		if StripANSI(previous.Output) != StripANSI(current.Output) {
			delta.BeforeLine, delta.AfterLine = firstDifference(previous.Output, current.Output)
			delta.BeforeLine, _ = Redact(delta.BeforeLine)
			delta.AfterLine, _ = Redact(delta.AfterLine)
		}
		deltas[index+1] = delta
	}
	return deltas
}

func firstDifference(before, after string) (string, string) {
	oldLines := strings.Split(strings.TrimSuffix(StripANSI(before), "\n"), "\n")
	newLines := strings.Split(strings.TrimSuffix(StripANSI(after), "\n"), "\n")
	limit := len(oldLines)
	if len(newLines) > limit {
		limit = len(newLines)
	}
	for index := 0; index < limit; index++ {
		oldLine, newLine := "(no line)", "(no line)"
		if index < len(oldLines) {
			oldLine = oldLines[index]
		}
		if index < len(newLines) {
			newLine = newLines[index]
		}
		if oldLine != newLine {
			return clip(strings.TrimSpace(oldLine), 160), clip(strings.TrimSpace(newLine), 160)
		}
	}
	return "", ""
}

func SuggestAgainstBaseline(source Report, deltas map[int]BaselineDelta) []Suggestion {
	base := SuggestCommands(source)
	reasons := map[int]string{}
	for _, suggestion := range base {
		reasons[suggestion.Number] = suggestion.Reason
	}
	for number, delta := range deltas {
		if delta.State == "regressed" || delta.State == "fixed" || delta.State == "exit changed" || delta.State == "output changed" {
			reasons[number] = fmt.Sprintf("%s versus baseline", delta.State)
		}
	}
	numbers := make([]int, 0, len(reasons))
	for number := range reasons {
		numbers = append(numbers, number)
	}
	sort.Ints(numbers)
	result := make([]Suggestion, 0, len(numbers))
	for _, number := range numbers {
		result = append(result, Suggestion{Number: number, Reason: reasons[number]})
	}
	return result
}
