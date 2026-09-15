package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shubhambhar007/proofshot/report"
)

const version = "0.4.0"

type commandsFlag []string

func (c *commandsFlag) String() string { return strings.Join(*c, ", ") }
func (c *commandsFlag) Set(value string) error {
	*c = append(*c, value)
	return nil
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "run":
		os.Exit(run(os.Args[2:]))
	case "render":
		os.Exit(render(os.Args[2:]))
	case "record":
		os.Exit(record(os.Args[2:]))
	case "share":
		os.Exit(share(os.Args[2:]))
	case "compare":
		os.Exit(compare(os.Args[2:]))
	case "verify":
		os.Exit(verify(os.Args[2:]))
	case "version", "--version", "-v":
		fmt.Println("proofshot", version)
	case "help", "--help", "-h":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func share(args []string) int {
	fs := flag.NewFlagSet("share", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	include := fs.String("include", "", "1-based command numbers or ranges, e.g. 1,3-5 (default: all)")
	context := fs.String("context", "", "short situation description for the recipient")
	question := fs.String("question", "", "what you need the recipient to answer")
	out := fs.String("out", "handoff.html", "single-file HTML output")
	list := fs.Bool("list", false, "show numbered commands without exporting")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "proofshot share needs one report.json file")
		return 2
	}
	source, err := report.ReadReport(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if source.Integrity != "" && !report.VerifyReport(source) {
		fmt.Fprintln(os.Stderr, "source report integrity mismatch")
		return 1
	}
	if *list {
		for index, command := range source.Commands {
			clean, _ := report.Redact(command.Command)
			fmt.Printf("%d. exit %d · %s\n", index+1, command.ExitCode, clean)
		}
		return 0
	}
	selected, err := selectCommands(*include, len(source.Commands))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	h, err := report.MakeHandoff(source, selected, *context, *question)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := h.WriteHTML(*out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "✓ %d command(s) exported → %s\n", len(h.Commands), *out)
	if len(h.Warnings) > 0 {
		fmt.Fprintf(os.Stderr, "Review before sending: %s may appear.\n", strings.Join(h.Warnings, ", "))
	}
	fmt.Fprintln(os.Stderr, "No upload occurred. Open the HTML file and inspect it before sharing.")
	return 0
}

func selectCommands(spec string, total int) ([]int, error) {
	if total == 0 {
		return nil, fmt.Errorf("source report contains no commands")
	}
	if strings.TrimSpace(spec) == "" {
		selected := make([]int, total)
		for i := range selected {
			selected[i] = i + 1
		}
		return selected, nil
	}
	seen := map[int]bool{}
	for _, part := range strings.Split(spec, ",") {
		bounds := strings.Split(strings.TrimSpace(part), "-")
		if len(bounds) == 0 || len(bounds) > 2 {
			return nil, fmt.Errorf("invalid command selection %q", part)
		}
		start, err := strconv.Atoi(strings.TrimSpace(bounds[0]))
		if err != nil {
			return nil, fmt.Errorf("invalid command selection %q", part)
		}
		end := start
		if len(bounds) == 2 {
			end, err = strconv.Atoi(strings.TrimSpace(bounds[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid command selection %q", part)
			}
		}
		if start < 1 || end > total || end < start {
			return nil, fmt.Errorf("command selection %q is outside 1..%d", part, total)
		}
		for i := start; i <= end; i++ {
			seen[i] = true
		}
	}
	selected := make([]int, 0, len(seen))
	for number := range seen {
		selected = append(selected, number)
	}
	sort.Ints(selected)
	return selected, nil
}

func record(args []string) int {
	fs := flag.NewFlagSet("record", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	out := fs.String("out", "proofshot-session", "output directory")
	title := fs.String("title", "Terminal session", "report title")
	cwd := fs.String("cwd", "", "starting working directory")
	noRedact := fs.Bool("no-redact", false, "disable automatic secret redaction")
	timeout := fs.Duration("timeout", 30*time.Minute, "timeout per command")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "proofshot record does not accept positional commands")
		return 2
	}
	workingDir := *cwd
	if workingDir == "" {
		var err error
		workingDir, err = os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}

	fmt.Fprintln(os.Stderr, "Proofshot recording started. Type commands normally; use exit or Ctrl-D to finish.")
	fmt.Fprintln(os.Stderr, "Tip: `cd` is preserved between commands.")
	reader := bufio.NewReader(os.Stdin)
	results := []report.CommandResult{}
	for {
		fmt.Fprintf(os.Stderr, "proofshot:%s$ ", displayDirectory(workingDir))
		line, err := reader.ReadString('\n')
		command := strings.TrimSpace(line)
		if command == "exit" || command == "quit" {
			break
		}
		if command != "" {
			if next, ok, cdErr := changeDirectory(command, workingDir); ok {
				result := report.CommandResult{Command: command, StartedAt: time.Now()}
				if cdErr != nil {
					result.ExitCode = 1
					result.Output = cdErr.Error() + "\n"
					fmt.Fprint(os.Stderr, result.Output)
				} else {
					workingDir = next
				}
				result.Findings = report.Analyze(result)
				results = append(results, result)
			} else {
				results = append(results, report.ExecuteLive(command, workingDir, *timeout, !*noRedact, nil, os.Stdout))
			}
		}
		if err != nil {
			if err != io.EOF {
				fmt.Fprintln(os.Stderr, "read command:", err)
			}
			break
		}
	}
	if len(results) == 0 {
		fmt.Fprintln(os.Stderr, "No commands captured; no report created.")
		return 0
	}
	r := report.New(*title, results)
	files, err := report.WriteBundle(r, *out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "write report:", err)
		return 1
	}
	printResult(r, files)
	return 0
}

func displayDirectory(directory string) string {
	home, err := os.UserHomeDir()
	if err == nil {
		if directory == home {
			return "~"
		}
		if strings.HasPrefix(directory, home+string(os.PathSeparator)) {
			return "~" + strings.TrimPrefix(directory, home)
		}
	}
	return directory
}

func changeDirectory(command, current string) (string, bool, error) {
	if command != "cd" && !strings.HasPrefix(command, "cd ") {
		return current, false, nil
	}
	target := strings.TrimSpace(strings.TrimPrefix(command, "cd"))
	if target == "" || target == "~" {
		home, err := os.UserHomeDir()
		return home, true, err
	}
	target = strings.Trim(target, "\"'")
	if strings.HasPrefix(target, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return current, true, err
		}
		target = filepath.Join(home, strings.TrimPrefix(target, "~/"))
	} else if !filepath.IsAbs(target) {
		target = filepath.Join(current, target)
	}
	resolved, err := filepath.Abs(target)
	if err != nil {
		return current, true, err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return current, true, fmt.Errorf("cd: %w", err)
	}
	if !info.IsDir() {
		return current, true, fmt.Errorf("cd: %s: not a directory", target)
	}
	return resolved, true, nil
}

func compare(args []string) int {
	fs := flag.NewFlagSet("compare", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	out := fs.String("out", "comparison.md", "comparison Markdown path")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 2 {
		fmt.Fprintln(os.Stderr, "proofshot compare needs two report.json files")
		return 2
	}
	before, err := report.ReadReport(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	after, err := report.ReadReport(fs.Arg(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	comparison := report.Compare(before, after)
	if err := os.WriteFile(*out, []byte(comparison.Markdown()), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "✓ compared %d command(s) → %s\n", len(comparison.Changes), *out)
	return 0
}

func verify(args []string) int {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "proofshot verify needs one report directory")
		return 2
	}
	if err := report.VerifyBundle(fs.Arg(0)); err != nil {
		fmt.Fprintln(os.Stderr, "verification failed:", err)
		return 1
	}
	r, err := report.ReadReport(fs.Arg(0) + string(os.PathSeparator) + "report.json")
	if err != nil || !report.VerifyReport(r) {
		fmt.Fprintln(os.Stderr, "verification failed: report integrity mismatch")
		return 1
	}
	fmt.Fprintln(os.Stderr, "✓ bundle checksums and report integrity verified")
	return 0
}

func run(args []string) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var commands commandsFlag
	fs.Var(&commands, "command", "command to run (repeatable; shorthand: -c)")
	fs.Var(&commands, "c", "command to run (repeatable)")
	out := fs.String("out", "proofshot-report", "output directory")
	title := fs.String("title", "Command report", "report title")
	cwd := fs.String("cwd", "", "working directory for commands")
	noRedact := fs.Bool("no-redact", false, "disable automatic secret redaction")
	timeout := fs.Duration("timeout", 5*time.Minute, "timeout per command")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	commands = append(commands, fs.Args()...)
	if len(commands) == 0 {
		fmt.Fprintln(os.Stderr, "proofshot run needs at least one command")
		return 2
	}

	results := make([]report.CommandResult, 0, len(commands))
	for index, command := range commands {
		fmt.Fprintf(os.Stderr, "[%d/%d] %s\n", index+1, len(commands), command)
		results = append(results, report.Execute(command, *cwd, *timeout, !*noRedact))
	}

	r := report.New(*title, results)
	files, err := report.WriteBundle(r, *out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "write report:", err)
		return 1
	}
	printResult(r, files)
	if r.Failed > 0 {
		return 1
	}
	return 0
}

func render(args []string) int {
	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	out := fs.String("out", "proofshot-report", "output directory")
	title := fs.String("title", "Captured output", "report title")
	noRedact := fs.Bool("no-redact", false, "disable automatic secret redaction")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "proofshot render needs at least one log file")
		return 2
	}
	results := make([]report.CommandResult, 0, fs.NArg())
	for _, path := range fs.Args() {
		result, err := report.FromFile(path, !*noRedact)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		results = append(results, result)
	}
	r := report.New(*title, results)
	files, err := report.WriteBundle(r, *out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "write report:", err)
		return 1
	}
	printResult(r, files)
	return 0
}

func printResult(r report.Report, files []string) {
	fmt.Fprintf(os.Stderr, "\n✓ %d command(s), %d failed, %s total\n", len(r.Commands), r.Failed, r.Duration.Round(time.Millisecond))
	for _, file := range files {
		fmt.Fprintln(os.Stderr, "  ", file)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `Proofshot — turn command runs into evidence, not screenshots.

Usage:
  proofshot run -c "npm test" -c "npm run build" [options]
  proofshot record [--title "Debugging session"] [--out proofshot-session]
  proofshot share --list proofshot-session/report.json
  proofshot share --include 1,3-5 --context "..." --question "..." proofshot-session/report.json
  proofshot run "npm test" "git status" [options]
  proofshot render app.log server.log [options]
  proofshot compare --out comparison.md before/report.json after/report.json
  proofshot verify proofshot-report

The output bundle contains failure findings, searchable HTML, paginated PNGs,
a Markdown summary, JSON, and an integrity manifest.`)
}
