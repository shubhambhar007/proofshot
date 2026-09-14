package report

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var locationPattern = regexp.MustCompile(`(?:^|[\s(])([^\s:()]+\.[A-Za-z0-9]+):(\d+)(?::\d+)?`)

var findingRules = []struct {
	kind    string
	pattern *regexp.Regexp
}{
	{"panic", regexp.MustCompile(`(?i)\b(panic|segmentation fault|stack overflow)\b`)},
	{"exception", regexp.MustCompile(`(?i)\b(exception|traceback|unhandled rejection)\b`)},
	{"error", regexp.MustCompile(`(?i)(^|\W)(error|fatal|failed|failure)(\W|$)`)},
	{"missing", regexp.MustCompile(`(?i)(not found|no such file|cannot find|can't resolve|module not found)`)},
	{"test", regexp.MustCompile(`(?i)(^|\s)(FAIL|FAILED)(\s|$)`)},
}

func Analyze(result CommandResult) []Finding {
	if result.ExitCode == 0 && !result.TimedOut {
		return nil
	}
	findings := []Finding{}
	seen := map[string]bool{}
	scanner := bufio.NewScanner(strings.NewReader(StripANSI(result.Output)))
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || len(findings) >= 5 {
			continue
		}
		for _, rule := range findingRules {
			if !rule.pattern.MatchString(line) {
				continue
			}
			evidence := clip(line, 220)
			if seen[evidence] {
				break
			}
			finding := Finding{Kind: rule.kind, Summary: summarize(rule.kind, line), Evidence: evidence, Line: lineNumber}
			if match := locationPattern.FindStringSubmatch(line); len(match) == 3 {
				finding.File = match[1]
				finding.Line, _ = strconv.Atoi(match[2])
			}
			findings = append(findings, finding)
			seen[evidence] = true
			break
		}
	}
	if result.TimedOut {
		findings = append([]Finding{{Kind: "timeout", Summary: "Command exceeded its time limit", Evidence: fmt.Sprintf("Stopped after %s", result.Duration)}}, findings...)
	}
	if len(findings) == 0 && result.ExitCode != 0 {
		findings = append(findings, Finding{Kind: "exit", Summary: fmt.Sprintf("Command exited with status %d", result.ExitCode), Evidence: lastNonEmptyLine(result.Output)})
	}
	return findings
}

func summarize(kind, line string) string {
	clean := strings.TrimSpace(line)
	if len([]rune(clean)) <= 100 {
		return clean
	}
	return strings.ToUpper(kind[:1]) + kind[1:] + " detected in command output"
}

func clip(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit-1]) + "…"
}

func lastNonEmptyLine(output string) string {
	lines := strings.Split(StripANSI(output), "\n")
	for index := len(lines) - 1; index >= 0; index-- {
		if value := strings.TrimSpace(lines[index]); value != "" {
			return clip(value, 220)
		}
	}
	return "No diagnostic output was produced."
}
