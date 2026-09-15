package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMatchBaselinePairsRepeatedCommandsAndRedactsSnippets(t *testing.T) {
	before := New("before", []CommandResult{
		{Command: "go test", Output: "first ok\n", ExitCode: 0},
		{Command: "go test", Output: "API_KEY=super-secret\nsecond ok\n", ExitCode: 0},
	})
	after := New("after", []CommandResult{
		{Command: "go test", Output: "first ok\n", ExitCode: 0},
		{Command: "go test", Output: "API_KEY=another-secret\napp.go:3: error: failed\n", ExitCode: 1},
	})
	deltas := MatchBaseline(before, after)
	if deltas[1].State != "same result" || deltas[2].State != "regressed" {
		t.Fatalf("unexpected deltas: %#v", deltas)
	}
	if strings.Contains(deltas[2].BeforeLine, "super-secret") || strings.Contains(deltas[2].AfterLine, "another-secret") {
		t.Fatal("secret leaked into comparison")
	}
	h, err := MakeHandoff(after, []int{2}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := h.AttachBaseline(before, deltas); err != nil {
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
	view := string(data)
	if !strings.Contains(view, "Versus baseline: regressed") || strings.Contains(view, "super-secret") {
		t.Fatalf("unexpected handoff: %s", view)
	}
}
