# Proofshot

Proofshot turns command runs into shareable evidence: one searchable HTML report, paginated PNGs that never cut off long output, and machine-readable JSON.

It is deliberately not another code-screenshot tool. Proofshot captures multiple commands as one report, records failures and timings, automatically redacts common secrets, and paginates output for tools that compress or reject extremely tall images.

## MVP features

- Run any number of commands in sequence
- Capture combined stdout and stderr, exit code, runtime, and timeout status
- Automatically redact API keys, bearer tokens, GitHub tokens, AWS access keys, and Stripe-style keys
- Create a searchable, printable HTML report
- Create numbered 1280×900 PNG pages for long output
- Create JSON for CI integrations and future hosted reports
- Render existing log files without executing their contents
- Return a failing process status if any captured command fails

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

Quoted positional commands work too:

```sh
proofshot run "go test ./..." "git status --short"
```

Render logs without executing anything:

```sh
proofshot render --title "Incident evidence" app.log worker.log
```

The default output is `proofshot-report/`:

```text
proofshot-report/
├── report.html
├── report.json
├── report-01.png
└── report-02.png
```

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
