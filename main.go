package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/shubhambhar007/proofshot/report"
)

const version = "0.1.0"

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
  proofshot run "npm test" "git status" [options]
  proofshot render app.log server.log [options]

The output bundle contains a searchable HTML report, paginated PNGs, and JSON.`)
}
