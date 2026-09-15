package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandoffSelectsRedactsAndEscapes(t *testing.T) {
	source := New("<script>alert(1)</script>", []CommandResult{
		{Command: "echo ok", Output: "ok\n"},
		{Command: "API_KEY=super-secret run", Output: "contact dev@example.com\napp.go:3: error: failed\n", ExitCode: 1},
	})
	h, err := MakeHandoff(source, []int{2}, "Need help", "Why did this fail?")
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Commands) != 1 || len(h.Warnings) != 1 {
		t.Fatalf("unexpected handoff: %#v", h)
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
	if strings.Contains(view, "super-secret") || strings.Contains(view, "<script>alert(1)</script>") || strings.Contains(view, "echo ok") {
		t.Fatal("secret, unsafe markup, or excluded command leaked")
	}
	if !strings.Contains(view, "[REDACTED]") || !strings.Contains(view, "app.go:3") {
		t.Fatal("expected redaction and failure evidence")
	}
}

func TestSuggestCommandsSurfacesFailureContextAndRecovery(t *testing.T) {
	source := New("investigation", []CommandResult{
		{Command: "setup", ExitCode: 0},
		{Command: "check config", ExitCode: 0},
		{Command: "build", ExitCode: 2, Output: "app.go:3: error: failed"},
		{Command: "inspect logs", ExitCode: 0},
		{Command: "cleanup", ExitCode: 0},
	})
	suggestions := SuggestCommands(source)
	want := []int{2, 3, 4}
	if len(suggestions) != len(want) {
		t.Fatalf("suggestions = %#v", suggestions)
	}
	for i, number := range want {
		if suggestions[i].Number != number {
			t.Fatalf("suggestions = %#v", suggestions)
		}
	}
	h, err := MakeHandoff(source, want, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.Summary, "first captured failure") || len(h.Steps) != 3 {
		t.Fatalf("summary/steps = %#v", h)
	}
}
