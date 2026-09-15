package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shubhambhar007/proofshot/report"
)

func TestDisplayDirectoryShowsPathAndShortensHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("~", "projects", "app")
	if got := displayDirectory(filepath.Join(home, "projects", "app")); got != want {
		t.Fatalf("displayDirectory() = %q, want %q", got, want)
	}
	if got := displayDirectory("/tmp/project"); got != "/tmp/project" {
		t.Fatalf("displayDirectory() = %q", got)
	}
}

func TestSelectCommandsKeepsOriginalOrder(t *testing.T) {
	selected, err := selectCommands("4,1,2-3,2", 4)
	if err != nil {
		t.Fatal(err)
	}
	for i, number := range selected {
		if number != i+1 {
			t.Fatalf("selected = %v", selected)
		}
	}
	if _, err := selectCommands("1,9", 4); err == nil {
		t.Fatal("expected out-of-range selection to fail")
	}
}

func TestRecheckPreviewDoesNotExecuteAndOptInCreatesComparison(t *testing.T) {
	directory := t.TempDir()
	source := report.New("shared", []report.CommandResult{{Command: "printf safe", Output: "old", ExitCode: 0}})
	sourceDir := filepath.Join(directory, "source")
	if _, err := report.WriteBundle(source, sourceDir); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(directory, "rechecked")
	path := filepath.Join(sourceDir, "report.json")
	if status := recheck([]string{"--out", out, path}); status != 0 {
		t.Fatalf("preview status %d", status)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("preview created an output directory: %v", err)
	}
	if status := recheck([]string{"--execute", "--out", out, path}); status != 0 {
		t.Fatalf("execution status %d", status)
	}
	if err := report.VerifyBundle(out); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(out, "recheck-handoff.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Versus baseline: output changed") {
		t.Fatal("comparison was missing")
	}
}
