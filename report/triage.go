package report

import (
	"fmt"
	"strings"
)

type TriageView struct {
	Excerpt   string
	LineCount int
	Long      bool
}

// TriageOutput preserves the full output elsewhere and extracts a small,
// numbered window around the first recognized diagnostic for quick reading.
func TriageOutput(command CommandResult) TriageView {
	plain := strings.TrimSuffix(StripANSI(command.Output), "\n")
	if plain == "" {
		return TriageView{}
	}
	lines := strings.Split(plain, "\n")
	view := TriageView{LineCount: len(lines), Long: len(lines) > 12}
	if !view.Long {
		return view
	}
	center := len(lines) - 1
	for _, finding := range command.Findings {
		needle := strings.TrimSpace(finding.Evidence)
		if needle == "" {
			continue
		}
		for index, line := range lines {
			if strings.Contains(strings.TrimSpace(line), needle) {
				center = index
				break
			}
		}
		if center != len(lines)-1 {
			break
		}
	}
	start, end := center-3, center+4
	if start < 0 {
		start = 0
	}
	if end > len(lines) {
		end = len(lines)
	}
	var excerpt strings.Builder
	for index := start; index < end; index++ {
		fmt.Fprintf(&excerpt, "%4d │ %s\n", index+1, lines[index])
	}
	view.Excerpt = excerpt.String()
	return view
}
