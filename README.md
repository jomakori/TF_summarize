# tf-summarize

A Go CLI that parses Terraform `plan` and `apply` output and produces a
beautified Markdown summaries — suitable for GitHub Actions step summaries or
PR comments.

## Usage

```bash
# Pipe parsing
terraform plan -no-color | tf-summarize
TF_PLAN_FILE=plan.txt tf-summarize

# JSON parsing (recommended)
terraform plan -out=plan.tfplan && terraform show -json plan.tfplan > plan.json
TF_PLAN_JSON=plan.json tf-summarize

# Apply phase
terraform apply -no-color -auto-approve 2>&1 | TF_PHASE=apply tf-summarize
```

## Environment Variables

| Variable             | Required | Default                  | Description                                               |
| -------------------- | -------- | ------------------------ | --------------------------------------------------------- |
| `TF_PLAN_FILE`       | No       | _(stdin)_                | Path to terraform plan/apply text output                  |
| `TF_PLAN_JSON`       | No       | —                        | Path to `terraform show -json` output (preferred)         |
| `TF_WORKSPACE`       | No       | `default`                | Workspace name shown in header                            |
| `TF_PHASE`           | No       | `plan`                   | `plan` or `apply`                                         |
| `TF_OUTPUT`          | No       | `stdout`                 | Output target(s): `stdout`, `gha`, `pr` (comma-separated) |
| `DESTROY`            | No       | `false`                  | `true`/`1` for destroy plans (red badge)                  |
| `GITHUB_TOKEN`       | For PR   | —                        | GitHub token for PR comments                              |
| `GITHUB_REPOSITORY`  | For PR   | —                        | `owner/repo` (auto-set in GHA)                            |
| `PR_NUMBER`          | For PR   | —                        | PR number to comment on                                   |
| `GITHUB_API_URL`     | No       | `https://api.github.com` | GitHub API base URL (for GHES)                            |
| `TF_EXIT_ON_CHANGES` | No       | `false`                  | Exit code 2 when changes detected                         |

## GitHub Actions Example

```yaml
- name: Summarize Plan
  if: always()
  env:
    TF_PLAN_JSON: plan.json
    TF_WORKSPACE: ${{ github.event.inputs.workspace || 'dev' }}
    TF_PHASE: plan
    TF_OUTPUT: gha,pr
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    PR_NUMBER: ${{ github.event.pull_request.number }}
  run: tf-summarize
```

## Output

### Change Type Badges

| Change Type    | Color  |
| -------------- | ------ |
| Create         | Green  |
| Modify         | Yellow |
| Remove/Replace | Red    |
| Import         | Purple |
| No Changes     | Blue   |
| Drift          | Orange |

### Phase Badges

| Phase           | Color | Condition       |
| --------------- | ----- | --------------- |
| Plan            | Blue  | Default         |
| Destroy         | Red   | `DESTROY=true`  |
| Apply (success) | Green | Apply succeeded |
| Apply (failed)  | Red   | Apply failed    |

### Scenarios

| Scenario         | Header                                         | Notes                                            |
| ---------------- | ---------------------------------------------- | ------------------------------------------------ |
| Plan w/ changes  | `Changes found for {workspace}`                | Caution alert shown if destroys/replaces present |
| Plan — no change | `No changes found for {workspace}`             | Single "No Changes" badge                        |
| Plan — destroy   | `Changes found for {workspace}`                | Phase badge red; `DESTROY=true`                  |
| Apply — success  | `Changes applied successfully for {workspace}` | Resource sections collapsible                    |
| Apply — failure  | `❌ Apply failed for {workspace}`              | Failed section open by default with errors       |

## Architecture

### Package Structure

```
├── main.go                  # Entry: flag/env parsing, parser selection, provider dispatch
├── internal/
│   ├── types.go             # Core types: Summary, ResourceChange, Action, Phase, OutputProvider
│   ├── utils.go             # Env helpers: GetEnv, GetEnvBool, RequireEnv
│   ├── hooks.go             # HookRegistry, HookEvent, HookFunc (lifecycle extension points)
│   ├── output.go            # WriteGHASummary, WritePRComment, FindPRForBranch
│   ├── parser/
│   │   ├── text.go          # Regex-based parsing of terraform stdout
│   │   └── json.go          # Structured parsing via terraform-json
│   ├── render/
│   │   ├── renderer.go      # Markdown generation from Summary
│   │   └── templates.go     # Badge/template strings
│   └── providers/
│       ├── base.go          # Shared no-op defaults
│       ├── stdout.go        # Writes to stdout
│       └── github.go        # Writes to GHA step summary or PR comment
└── tests/
    ├── parser_test.go        # Text parser: plan/apply/destroy/drift/no-changes
    ├── parser_cidr_block_test.go  # CIDR edge cases for text parser
    ├── json_parser_test.go   # JSON parser: all actions + invalid input
    ├── renderer_test.go      # Renderer: all plan/apply scenarios
    └── github_provider_test.go    # GitHub provider: GHA + PR comment output
```

### Data Flow

```
Input (stdin | TF_PLAN_FILE | TF_PLAN_JSON)
  → parser/{text,json}.go  →  internal.Summary
  → render/renderer.go     →  Markdown string
  → providers/*.go         →  OutputProvider.WriteSummary()
```

### Key Types (`internal/types.go`)

| Type             | Values                                                               | Purpose                         |
| ---------------- | -------------------------------------------------------------------- | ------------------------------- |
| `Phase`          | `plan` \| `apply`                                                    | Controls header + parsing logic |
| `Action`         | `create` \| `update` \| `destroy` \| `replace` \| `read` \| `import` | Resource change classification  |
| `OutputTarget`   | `stdout` \| `gha` \| `pr`                                            | Where output is written         |
| `ResourceChange` | `{Address, Action, Success, Error, Timestamp}`                       | Single affected resource        |
| `Summary`        | counts + resource lists + errors + warnings + metadata               | Central parsed result           |

### OutputProvider Interface

```go
type OutputProvider interface {
    WriteSummary(summary *Summary, markdown string) error
    WriteOutputs(summary *Summary, markdown string) error
    Name() string
}
```

To add a provider: implement this interface, embed `BaseProvider`, register in the `switch` in `main.go`.

### Parsing Strategy

| Env Var        | Parser    | Notes                         |
| -------------- | --------- | ----------------------------- |
| `TF_PLAN_JSON` | `json.go` | Preferred — accurate counts   |
| `TF_PLAN_FILE` | `text.go` | Saved stdout fallback         |
| stdin          | `text.go` | Default; required for `apply` |

### Exit Codes

| Code | Meaning                                                |
| ---- | ------------------------------------------------------ |
| `0`  | Success, no errors                                     |
| `1`  | Terraform errors or failures detected                  |
| `2`  | Changes detected (only when `TF_EXIT_ON_CHANGES=true`) |

## Build & Test

```bash
make build                              # dev build
make build-versioned VERSION=1.0.0     # with version injection
go test ./...                          # run all tests
gofmt -w .                             # format before committing
```

## Releases

Automated via GitHub Actions using [conventional commits](https://www.conventionalcommits.org/):

| Prefix                       | Bump       |
| ---------------------------- | ---------- |
| `feat:`                      | Minor      |
| `fix:`, `perf:`, `refactor:` | Patch      |
| `feat!:` / `BREAKING CHANGE` | Major      |
| `docs:`, `chore:`, `test:`   | No release |

```bash
go install github.com/jomakori/TF_summarize@latest
```

## Contributing

### Prerequisites
- Go 1.24.2+

### Setup
```bash
go mod download
make build
```

**Commit format** — uses [conventional commits](https://www.conventionalcommits.org/) to drive automated releases

### Guidelines

- Tests — all code additions or tweaks must include tests; run `go test ./...` before opening a PR
- Comments — only add comments where context is genuinely unclear; keep them concise single-liners
- DRY & KISS — keep code simple and avoid duplication; fix existing implementations rather than layering on top of them
