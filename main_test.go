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
