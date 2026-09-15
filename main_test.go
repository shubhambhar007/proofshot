package main

import (
	"os"
	"path/filepath"
	"testing"
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
