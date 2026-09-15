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
