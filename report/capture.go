package report

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

type lockedBuffer struct {
	mu   sync.Mutex
	b    bytes.Buffer
	live io.Writer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n, err := b.b.Write(p)
	if b.live != nil {
		if _, liveErr := b.live.Write(p); err == nil {
			err = liveErr
		}
	}
	return n, err
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
}

func Execute(command, cwd string, timeout time.Duration, redact bool) CommandResult {
	return execute(command, cwd, timeout, redact, nil, nil)
}

// ExecuteLive captures a command while mirroring its output to the terminal.
func ExecuteLive(command, cwd string, timeout time.Duration, redact bool, stdin io.Reader, live io.Writer) CommandResult {
	return execute(command, cwd, timeout, redact, stdin, live)
}

func execute(command, cwd string, timeout time.Duration, redact bool, stdin io.Reader, live io.Writer) CommandResult {
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	shell, shellArg := "/bin/sh", "-lc"
	if runtime.GOOS == "windows" {
		shell, shellArg = "cmd.exe", "/C"
	}
	cmd := exec.CommandContext(ctx, shell, shellArg, command)
	if cwd != "" {
		cmd.Dir = cwd
	}
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "CLICOLOR_FORCE=1", "FORCE_COLOR=1")
	output := lockedBuffer{live: live}
	cmd.Stdin = stdin
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		exitCode = 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}
	text := output.String()
	redactions := 0
	if redact {
		text, redactions = Redact(text)
	}
	result := CommandResult{
		Command: command, Output: text, ExitCode: exitCode,
		Duration: time.Since(started), StartedAt: started,
		TimedOut: ctx.Err() == context.DeadlineExceeded, Redactions: redactions,
	}
	result.Findings = Analyze(result)
	return result
}

func FromFile(path string, redact bool) (CommandResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return CommandResult{}, fmt.Errorf("read %s: %w", path, err)
	}
	text, count := string(data), 0
	if redact {
		text, count = Redact(text)
	}
	result := CommandResult{Command: path, Output: text, ExitCode: 0, StartedAt: time.Now(), Redactions: count}
	result.Findings = Analyze(result)
	return result, nil
}

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(authorization\s*:\s*bearer\s+)[A-Za-z0-9._~+/=-]+`),
	regexp.MustCompile(`(?i)((?:api[_-]?key|token|secret|password|passwd|pwd)\s*[=:]\s*)[^\s"']+`),
	regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,})\b`),
	regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
	regexp.MustCompile(`\b(?:sk|pk)_(?:live|test)_[A-Za-z0-9]{16,}\b`),
}

func Redact(input string) (string, int) {
	count := 0
	for index, pattern := range secretPatterns {
		input = pattern.ReplaceAllStringFunc(input, func(match string) string {
			count++
			if index < 2 {
				parts := regexp.MustCompile(`[=:]\s*`).Split(match, 2)
				prefixEnd := len(match) - len(parts[len(parts)-1])
				return match[:prefixEnd] + "[REDACTED]"
			}
			return "[REDACTED]"
		})
	}
	return input, count
}

var ansiPattern = regexp.MustCompile(`\x1b(?:\[[0-?]*[ -/]*[@-~]|\][^\x07]*(?:\x07|\x1b\\))`)

func StripANSI(input string) string {
	input = ansiPattern.ReplaceAllString(input, "")
	return strings.ReplaceAll(input, "\r\n", "\n")
}
