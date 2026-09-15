package report

import (
	"fmt"
	"html/template"
	"os"
	"regexp"
	"strings"
	"time"
)

type Handoff struct {
	Title           string
	Context         string
	Question        string
	Summary         string
	Commands        []CommandResult
	Steps           []HandoffStep
	SourceHash      string
	BaselineTitle   string
	BaselineHash    string
	BaselineSummary string
	Warnings        []string
}

type HandoffStep struct {
	Number  int
	Reason  string
	Command CommandResult
	Delta   *BaselineDelta
	Triage  TriageView
}

var privacyPatterns = []struct {
	name    string
	pattern *regexp.Regexp
}{
	{"email addresses", regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`)},
	{"private network addresses", regexp.MustCompile(`\b(?:10|192\.168|172\.(?:1[6-9]|2[0-9]|3[01]))\.\d{1,3}(?:\.\d{1,3})?\b`)},
	{"home-directory paths", regexp.MustCompile(`(?:/Users/|/home/)[^\s/]+/`)},
}

func MakeHandoff(source Report, selected []int, context, question string) (Handoff, error) {
	if source.Integrity != "" && !VerifyReport(source) {
		return Handoff{}, fmt.Errorf("source report integrity mismatch")
	}
	if len(selected) == 0 {
		return Handoff{}, fmt.Errorf("select at least one command")
	}
	cleanContext, _ := Redact(context)
	cleanQuestion, _ := Redact(question)
	cleanTitle, _ := Redact(source.Title)
	h := Handoff{Title: cleanTitle, Context: cleanContext, Question: cleanQuestion, SourceHash: source.Integrity}
	seen := map[int]bool{}
	warnings := map[string]bool{}
	recommendations := map[int]string{}
	for _, suggestion := range SuggestCommands(source) {
		recommendations[suggestion.Number] = suggestion.Reason
	}
	for _, number := range selected {
		if number < 1 || number > len(source.Commands) {
			return Handoff{}, fmt.Errorf("command %d is outside 1..%d", number, len(source.Commands))
		}
		if seen[number] {
			continue
		}
		seen[number] = true
		original := source.Commands[number-1]
		command := original
		var count int
		command.Command, count = Redact(command.Command)
		command.Redactions += count
		command.Output, count = Redact(command.Output)
		command.Redactions += count
		command.Findings = Analyze(command)
		for _, item := range privacyPatterns {
			if item.pattern.MatchString(command.Command) || item.pattern.MatchString(command.Output) {
				warnings[item.name] = true
			}
		}
		h.Commands = append(h.Commands, command)
		reason := recommendations[number]
		if reason == "" {
			reason = "Selected by sender"
		}
		h.Steps = append(h.Steps, HandoffStep{Number: number, Reason: reason, Command: command, Triage: TriageOutput(command)})
	}
	h.Summary = SummarizeHandoff(h.Commands)
	for _, item := range privacyPatterns {
		if item.pattern.MatchString(h.Title) || item.pattern.MatchString(h.Context) || item.pattern.MatchString(h.Question) {
			warnings[item.name] = true
		}
		if warnings[item.name] {
			h.Warnings = append(h.Warnings, item.name)
		}
	}
	return h, nil
}

func (h *Handoff) AttachBaseline(before Report, deltas map[int]BaselineDelta) error {
	if before.Integrity != "" && !VerifyReport(before) {
		return fmt.Errorf("baseline report integrity mismatch")
	}
	h.BaselineTitle, _ = Redact(before.Title)
	h.BaselineHash = before.Integrity
	regressed, fixed, changed := 0, 0, 0
	for index := range h.Steps {
		step := &h.Steps[index]
		delta, found := deltas[step.Number]
		if !found {
			continue
		}
		step.Delta = &delta
		if delta.State == "regressed" || delta.State == "fixed" || delta.State == "exit changed" || delta.State == "output changed" {
			step.Reason = delta.State + " versus baseline"
		}
		switch delta.State {
		case "regressed":
			regressed++
		case "fixed":
			fixed++
		case "output changed", "exit changed":
			changed++
		}
		for _, item := range privacyPatterns {
			if item.pattern.MatchString(delta.BeforeLine) || item.pattern.MatchString(delta.AfterLine) {
				h.Warnings = appendWarning(h.Warnings, item.name)
			}
		}
	}
	h.BaselineSummary = fmt.Sprintf("Among the selected steps: %d regressed, %d fixed, and %d changed compared with `%s`. This is observed difference, not a cause.", regressed, fixed, changed, h.BaselineTitle)
	for _, item := range privacyPatterns {
		if item.pattern.MatchString(h.BaselineTitle) {
			h.Warnings = appendWarning(h.Warnings, item.name)
		}
	}
	return nil
}

func appendWarning(warnings []string, name string) []string {
	for _, existing := range warnings {
		if existing == name {
			return warnings
		}
	}
	return append(warnings, name)
}

func (h Handoff) WriteHTML(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return handoffTemplate.Execute(f, h)
}

var handoffTemplate = template.Must(template.New("handoff").Funcs(template.FuncMap{
	"duration": func(d time.Duration) string { return d.Round(time.Millisecond).String() },
	"plain":    StripANSI,
	"join":     strings.Join,
}).Parse(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{{.Title}} · Proofshot handoff</title><style>
:root{color-scheme:dark;--bg:#090b10;--card:#11151d;--line:#293241;--text:#e8edf5;--muted:#9da9ba;--red:#ff8080;--green:#58d98c;--accent:#a6b8ff}*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--text);font:14px/1.55 ui-monospace,SFMono-Regular,Menlo,monospace}main{max-width:1050px;margin:auto;padding:48px 24px 80px}h1{font:700 38px/1.15 system-ui;margin:5px 0 24px}.eyebrow{color:var(--accent);text-transform:uppercase;letter-spacing:.12em;font-size:11px;font-weight:700}.intro,.warning,.command{background:var(--card);border:1px solid var(--line);border-radius:14px;padding:20px;margin:15px 0}.intro{font:16px/1.5 system-ui}.label{font:700 11px/1.2 ui-monospace;text-transform:uppercase;letter-spacing:.1em;color:var(--muted);margin:0 0 8px}.intro p{margin:0 0 18px}.intro p:last-child{margin:0}.warning{border-color:#8b5e26;background:#241b11;color:#ffcf83}.warning strong{display:block;margin-bottom:5px}.command{padding:0;overflow:hidden}.command-head{padding:16px 20px;display:flex;gap:16px;align-items:start;border-bottom:1px solid var(--line)}.status{color:var(--green)}.failed .status{color:var(--red)}code{overflow-wrap:anywhere}.meta{margin-left:auto;color:var(--muted);white-space:nowrap}.reason{padding:7px 20px;color:var(--accent);border-bottom:1px solid var(--line)}.delta{padding:10px 20px;background:#151b27;border-bottom:1px solid var(--line)}.delta strong{color:var(--accent)}details{border-top:1px solid var(--line)}summary{cursor:pointer;padding:12px 20px;color:var(--accent)}.excerpt{padding:16px 20px;background:#111a20;white-space:pre-wrap;overflow-wrap:anywhere}.before,.after{display:block;overflow-wrap:anywhere}.finding{padding:9px 20px;background:#22161b;color:#ffc5c5}.finding b{color:var(--red)}pre{margin:0;padding:20px;max-height:65vh;overflow:auto;white-space:pre-wrap;overflow-wrap:anywhere;background:#0c1017}footer{color:var(--muted);font-size:12px;margin-top:32px;overflow-wrap:anywhere}@media print{body{background:#fff;color:#111}.intro,.warning,.command{break-inside:avoid;border-color:#ddd;background:#fff;color:#111}pre{max-height:none;background:#f6f6f6;color:#111}}</style></head><body><main><div class="eyebrow">Proofshot · Troubleshooting handoff</div><h1>{{.Title}}</h1><section class="intro">{{if .Context}}<div class="label">Context</div><p>{{.Context}}</p>{{end}}<div class="label">What happened</div><p>{{.Summary}}</p>{{if .BaselineTitle}}<div class="label">What changed</div><p>{{.BaselineSummary}}</p>{{end}}{{if .Question}}<div class="label">What I need help with</div><p>{{.Question}}</p>{{end}}<div class="label">Evidence</div><p>{{len .Commands}} selected commands, kept in their original order.</p></section>{{if .Warnings}}<section class="warning"><strong>Review before sending</strong>Potentially identifying content detected: {{join .Warnings ", "}}. Automatic redaction is best-effort.</section>{{end}}{{range .Steps}}<section class="command {{if .Command.ExitCode}}failed{{end}}"><div class="command-head"><span class="status">{{if .Command.ExitCode}}✗{{else}}✓{{end}}</span><code>$ {{.Command.Command}}</code><span class="meta">step {{.Number}} · exit {{.Command.ExitCode}} · {{duration .Command.Duration}}</span></div><div class="reason">Why included: {{.Reason}}</div>{{if .Delta}}<div class="delta"><strong>Versus baseline: {{.Delta.State}}</strong> · exit {{.Delta.BeforeExit}} → {{.Delta.AfterExit}}{{if .Delta.BeforeLine}}<span class="before">Before: {{.Delta.BeforeLine}}</span><span class="after">Now: {{.Delta.AfterLine}}</span>{{end}}</div>{{end}}{{range .Command.Findings}}<div class="finding"><b>{{.Kind}}</b> · {{.Summary}}</div>{{end}}{{if .Triage.Long}}<div class="reason">Failure-first excerpt · {{.Triage.LineCount}} total lines; full evidence preserved below</div><pre class="excerpt">{{.Triage.Excerpt}}</pre><details><summary>Show full output ({{.Triage.LineCount}} lines)</summary><pre>{{plain .Command.Output}}</pre></details>{{else}}<pre>{{if .Command.Output}}{{plain .Command.Output}}{{else}}(no output){{end}}</pre>{{end}}</section>{{end}}<footer>Generated locally; no upload occurred. Source report digest: {{.SourceHash}}. {{if .BaselineHash}}Baseline digest: {{.BaselineHash}}.{{end}} Suggestions show sequence, not proven causality. Review this file before sharing.</footer></main></body></html>`))
