package report

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTriageOutputFindsDiagnosticWithoutCuttingFullLog(t *testing.T) {
	var output strings.Builder
	for i := 1; i <= 80; i++ {
		if i == 42 {
			output.WriteString("app.go:7: error: broken\n")
		} else {
			fmt.Fprintf(&output, "noise %d\n", i)
		}
	}
	command := CommandResult{ExitCode: 1, Output: output.String()}
	command.Findings = Analyze(command)
	view := TriageOutput(command)
	if !view.Long || view.LineCount != 80 || !strings.Contains(view.Excerpt, "42 │ app.go:7: error: broken") || strings.Contains(view.Excerpt, "noise 1") {
		t.Fatalf("unexpected triage view: %#v", view)
	}
	if !strings.Contains(command.Output, "noise 1") {
		t.Fatal("full evidence was altered")
	}
	source := New("huge failure", []CommandResult{command})
	h, err := MakeHandoff(source, []int{1}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "handoff.html")
	if err := h.WriteHTML(path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Show full output (80 lines)") || !strings.Contains(string(data), "noise 1") {
		t.Fatal("full output not preserved behind triage view")
	}
}
