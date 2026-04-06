# tfplan-summary

A small Go CLI that parses Terraform `plan` and `apply` output and produces a
beautified Markdown summary — suitable for GitHub Actions step summaries or
PR comments.

## Usage

```bash
# Pipe terraform plan output
terraform plan -no-color | tfplan-summary

# Or point to a saved output file
terraform plan -no-color -out=plan.out > plan.txt
TF_PLAN_FILE=plan.txt tfplan-summary

# Apply phase
terraform apply -no-color -auto-approve 2>&1 | TF_PHASE=apply tfplan-summary
```

## Environment Variables

| Variable              | Required | Default   | Description                                                  |
| --------------------- | -------- | --------- | ------------------------------------------------------------ |
| `TF_PLAN_FILE`        | No       | _(stdin)_ | Path to file containing terraform plan/apply text output     |
| `TF_WORKSPACE`        | No       | `default` | Workspace name shown in the header (falls back to `WORKSPACE`) |
| `TF_PHASE`            | No       | `plan`    | `plan` or `apply` — controls header messaging and parsing    |
| `TF_OUTPUT`           | No       | `stdout`  | Output target(s): `stdout`, `gha`, `pr` (comma-separated)   |
| `GITHUB_TOKEN`        | For PR   | —         | GitHub token for posting PR comments                         |
| `GITHUB_REPOSITORY`   | For PR   | —         | `owner/repo` — set automatically in GHA                      |
| `PR_NUMBER`           | For PR   | —         | Pull request number to comment on                            |
| `GITHUB_API_URL`      | No       | `https://api.github.com` | GitHub API base URL (for GHES)              |
| `TF_EXIT_ON_CHANGES`  | No       | `false`   | Exit code 2 when changes are detected (useful for CI gates)  |

## GitHub Actions Example

```yaml
jobs:
  plan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Terraform Plan
        id: plan
        run: terraform plan -no-color 2>&1 | tee plan.txt

      - name: Summarize Plan
        if: always()
        env:
          TF_PLAN_FILE: plan.txt
          TF_WORKSPACE: ${{ github.event.inputs.workspace || 'dev' }}
          TF_PHASE: plan
          TF_OUTPUT: gha,pr
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          PR_NUMBER: ${{ github.event.pull_request.number }}
        run: tfplan-summary

  apply:
    runs-on: ubuntu-latest
    needs: plan
    steps:
      - uses: actions/checkout@v4

      - name: Terraform Apply
        run: terraform apply -no-color -auto-approve 2>&1 | tee apply.txt

      - name: Summarize Apply
        if: always()
        env:
          TF_PLAN_FILE: apply.txt
          TF_WORKSPACE: prod
          TF_PHASE: apply
          TF_OUTPUT: gha,pr
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          PR_NUMBER: ${{ github.event.pull_request.number }}
        run: tfplan-summary
```

## Output

### Plan — Create

```
## 📋 Changes found for `plat-ue2-sandbox`

![PLAN](badge) ![CREATE](badge)

**Plan:** **9** to add

▸ Create (9)          — collapsible resource list
▸ Terraform Plan Output — collapsible full plan diff
```

### Plan — Replace (with destroy warning)

```
## 📋 Changes found for `plat-ue2-sandbox`

![PLAN](badge) ![REPLACE](badge)

> [!CAUTION]
> **Terraform will delete resources!**

**Plan:** **1** to add, **1** to change, **2** to destroy

▸ Update (1)
▸ Destroy (1)
▸ Replace (1)
▸ Terraform Plan Output
```

### Apply — Success

```
## ✅ Changes applied successfully for `prod`

![APPLY](badge) ![CREATE](badge)

**Result:** **3** added

▸ ✅ Created (3)       — each resource listed
▸ Terraform Apply Output
```

### Apply — Partial Failure

```
## ❌ Apply failed for `prod`

![APPLY](badge) ![1 FAILED](badge)

**Result:** **1** failed

▸ ✅ Created (2)       — resources that succeeded
▸ ❌ Failed (1)        — open by default, shows resource + error
▸ Terraform Apply Output
```

### No Changes

```
## ✅ No changes found for `dev`

![PLAN](badge) ![NO CHANGES](badge)

Infrastructure is up-to-date. No changes needed.
```

## Build

```bash
go build -o tfplan-summary ./cmd/tfplan-summary
```

## Test

```bash
go test ./...
```
