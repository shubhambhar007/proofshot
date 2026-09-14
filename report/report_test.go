package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRedact(t *testing.T) {
	input := "API_KEY=super-secret\nAuthorization: Bearer abc.def.ghi\nghp_abcdefghijklmnopqrstuvwxyz"
	output, count := Redact(input)
	if count != 3 {
		t.Fatalf("got %d redactions, want 3", count)
	}
	for _, secret := range []string{"super-secret", "abc.def.ghi", "ghp_abcdefghijklmnopqrstuvwxyz"} {
		if strings.Contains(output, secret) {
			t.Fatalf("secret remained: %s", secret)
		}
	}
}

func TestExecuteCapturesFailure(t *testing.T) {
	r := Execute("printf hello; printf problem >&2; exit 7", "", time.Second, true)
	if r.ExitCode != 7 {
		t.Fatalf("exit code = %d", r.ExitCode)
	}
	if !strings.Contains(r.Output, "hello") || !strings.Contains(r.Output, "problem") {
		t.Fatalf("missing output: %q", r.Output)
	}
}

func TestWriteBundlePaginates(t *testing.T) {
	dir := t.TempDir()
	long := strings.Repeat("a long output line\n", 120)
	r := New("Test", []CommandResult{{Command: "demo", Output: long}})
	files, err := WriteBundle(r, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < 5 {
		t.Fatalf("expected html, json, and multiple PNGs; got %v", files)
	}
	for _, name := range []string{"report.html", "report.json", "report-01.png"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
}
