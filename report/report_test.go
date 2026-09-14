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
	for _, name := range []string{"report.html", "report.json", "summary.md", "SHA256SUMS", "report-01.png"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAnalyzeFindsCauseAndLocation(t *testing.T) {
	result := CommandResult{ExitCode: 1, Output: "src/app.ts:42:7 error: Cannot find module widget\n"}
	findings := Analyze(result)
	if len(findings) == 0 || findings[0].File != "src/app.ts" || findings[0].Line != 42 {
		t.Fatalf("unexpected findings: %#v", findings)
	}
}

func TestVerifyBundleDetectsChanges(t *testing.T) {
	dir := t.TempDir()
	r := New("Test", []CommandResult{{Command: "demo", Output: "ok\n"}})
	if _, err := WriteBundle(r, dir); err != nil {
		t.Fatal(err)
	}
	if err := VerifyBundle(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "report.html"), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifyBundle(dir); err == nil {
		t.Fatal("expected modified report to fail verification")
	}
}

func TestCompareDetectsRegression(t *testing.T) {
	before := New("before", []CommandResult{{Command: "go test ./...", ExitCode: 0, Output: "ok\n"}})
	after := New("after", []CommandResult{{Command: "go test ./...", ExitCode: 1, Output: "FAIL\n"}})
	comparison := Compare(before, after)
	if len(comparison.Changes) != 1 || comparison.Changes[0].State != "regressed" {
		t.Fatalf("unexpected comparison: %#v", comparison)
	}
}
