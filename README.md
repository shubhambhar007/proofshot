# Proofshot

Proofshot turns command runs into shareable, verifiable evidence. It extracts the lines that explain a failure, creates paginated images that never cut output off, and compares one run with another.

It is deliberately not another code-screenshot tool. Proofshot captures multiple commands as one report, records failures and timings, automatically redacts common secrets, and paginates output for tools that compress or reject extremely tall images.

## MVP features

- Run any number of commands in sequence
- Record an unplanned, command-by-command terminal session with live output
- Capture combined stdout and stderr, exit code, runtime, and timeout status
- Automatically redact API keys, bearer tokens, GitHub tokens, AWS access keys, and Stripe-style keys
- Create a searchable, printable HTML report
- Create numbered 1280×900 PNG pages for long output
- Create JSON for CI integrations and future hosted reports
- Render existing log files without executing their contents
- Return a failing process status if any captured command fails
- Pull likely root-cause lines and file locations out of noisy failures
- Generate a PR/issue-ready Markdown summary
- Detect regressions, fixes, duration changes, and output-size changes between runs
- Verify every artifact in a bundle with SHA-256 checksums
- Curate a standalone troubleshooting handoff from selected commands after recording

## Install

Requires Go 1.24 or newer.

Once the repository is published, users will be able to install it with `go install`. For now, build the local MVP:

For local development:

```sh
go build -o proofshot .
```

## Usage

Capture several commands:

```sh
proofshot run \
  -c "npm test" \
  -c "npm run build" \
  -c "git status" \
  --title "Release verification"
```

Record commands interactively, then type `exit` or press Ctrl-D to create the report:

```sh
proofshot record --title "Debugging checkout failure" --out checkout-debug

proofshot:~/projects/app$ npm test
proofshot:~/projects/app$ git status
proofshot:~/projects/app$ cd ../another-project
proofshot:~/projects/another-project$ go test ./...
proofshot:~/projects/another-project$ exit
```

The prompt shows the current working directory (shortening your home directory to `~`). Commands execute in sequence with output shown live and captured simultaneously. Directory changes made with `cd` persist during the recording and update the prompt immediately. The recorder is intentionally command-oriented; full-screen programs such as Vim and interactive password prompts are not supported in this MVP.

After recording, choose only the relevant steps and add the recipient's question:

```sh
proofshot share --list checkout-debug/report.json
proofshot share --suggest checkout-debug/report.json
proofshot share \
  --include 1,3-5 \
  --context "Checkout fails after the dependency update" \
  --question "Which failing step should we fix first?" \
  --out checkout-handoff.html \
  checkout-debug/report.json
```

The handoff is one self-contained HTML file, with live output preserved in original command order. Proofshot reruns secret redaction on both command text and output and warns about identifying content such as email addresses, local paths, and private network addresses. This is a review aid, not a guarantee; open the file and inspect it before sending. No hosted upload happens.

Without `--include`, Proofshot recommends a concise evidence subset: failed commands, the step immediately before the first failure, and the first successful step afterward. Each recommendation says why it was included. The handoff also writes a factual summary of the selected results. These are sequence-based heuristics, not an AI diagnosis or proof that the preceding command caused the failure. Use `--all` to export every command or `--include` to override the recommendation.

For a “works on my machine” handoff, attach an earlier run:

```sh
proofshot share \
  --baseline baseline/report.json \
  --question "Why does this fail now?" \
  --out regression-handoff.html \
  current/report.json
```

The default selection then prioritizes regressions, fixes, changed exits, and changed output. Each selected step shows its earlier and current exit status plus the first differing output line. Matching uses command text and occurrence order, so repeated commands remain distinct. The difference is factual evidence, not a diagnosis. The baseline snippets are redacted again and the handoff still requires a privacy review before sending.

## Two-way troubleshooting

Large outputs open on a numbered excerpt near the first detected diagnostic; the full, uncut output is available via **Show full output**. This helps a recipient reach the signal without losing the evidence trail.

A recipient with the project can preview the suggested checks and selectively run them on their machine:

```sh
proofshot recheck shared/report.json
proofshot recheck --include 1,3 --cwd /path/to/project --execute shared/report.json
```

The first command only previews—it never executes. `--execute` is an explicit opt-in after reviewing every command, which may have side effects. Recheck creates a new bundle and `recheck-handoff.html` showing earlier/current results and first differing lines. It refuses commands containing redacted secrets and `cd` steps. Reports have checksums but no identity signature: accept a report only from a trusted person, inspect its commands, and use an appropriate environment before running them. The generated files remain local.

Quoted positional commands work too:

```sh
proofshot run "go test ./..." "git status --short"
```

Render logs without executing anything:

```sh
proofshot render --title "Incident evidence" app.log worker.log
```

Verify that a bundle has not changed since capture:

```sh
proofshot verify proofshot-report
```

Compare a baseline with a new run:

```sh
proofshot compare --out comparison.md baseline/report.json proofshot-report/report.json
```

The default output is `proofshot-report/`:

```text
proofshot-report/
├── report.html
├── report.json
├── summary.md
├── SHA256SUMS
├── report-01.png
└── report-02.png
```

## Why this is more than a log file

A log preserves text. Proofshot preserves the result: command boundaries, exit codes, timings, redactions, probable failure causes, shareable visual pages, and a machine-readable history that can answer “what changed?” The HTML remains searchable and the Markdown summary can be pasted directly into a pull request or incident ticket.

Checksum verification detects files modified after capture when the manifest is trusted. Cryptographic signing and hosted provenance are natural paid-tier follow-ups; the current local integrity check is not an identity signature.

## Safety

Redaction is enabled by default. It is best-effort, so always review a report before sharing it publicly. `--no-redact` is available for controlled environments.

`render` only reads logs. `share` reads reports and writes a handoff without executing their contents. `run` and `record` execute shell commands; `recheck` executes report commands only with the explicit `--execute` flag.

## Product direction

The strongest commercial wedge is trustworthy sharing rather than prettier pixels:

1. CI uploads with stable report URLs
2. Team workspaces and retention controls
3. PR/issue comments with concise failure summaries
4. Organization-level redaction policies and audit logs
5. Private, self-hosted report storage

Advertising fits a free hosted public-report tier, but paid private reports, team history, and CI integrations are likely stronger revenue paths. An acquisition target would more naturally be a developer-tooling, observability, CI, or documentation company.

## License

Copyright retained. Choose a license before publishing the repository.
