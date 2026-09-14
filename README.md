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

proofshot:project$ npm test
proofshot:project$ git status
proofshot:project$ exit
```

Commands execute in sequence with output shown live and captured simultaneously. Directory changes made with `cd` persist during the recording. The recorder is intentionally command-oriented; full-screen programs such as Vim and interactive password prompts are not supported in this MVP.

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

`render` only reads files. It never executes their contents. `run` is the only command that executes shell input.

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
