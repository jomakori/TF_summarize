# tf-summarize

A small Go CLI that parses Terraform `plan` and `apply` output and produces a
beautified Markdown summary — suitable for GitHub Actions step summaries or
PR comments.

## Usage

```bash
# Pipe terraform plan output
terraform plan -no-color | tf-summarize

# Or point to a saved output file
terraform plan -no-color -out=plan.out > plan.txt
TF_PLAN_FILE=plan.txt tf-summarize

# Apply phase
terraform apply -no-color -auto-approve 2>&1 | TF_PHASE=apply tf-summarize
```

## Environment Variables

| Variable              | Required | Default   | Description                                                  |
| --------------------- | -------- | --------- | ------------------------------------------------------------ |
| `TF_PLAN_FILE`        | No       | _(stdin)_ | Path to file containing terraform plan/apply text output     |
| `TF_WORKSPACE`        | No       | `default` | Workspace name shown in the header (falls back to `WORKSPACE`) |
| `TF_PHASE`            | No       | `plan`    | `plan` or `apply` — controls header messaging and parsing    |
| `TF_OUTPUT`           | No       | `stdout`  | Output target(s): `stdout`, `gha`, `pr` (comma-separated)   |
| `DESTROY`             | No       | `false`   | Set to `true` or `1` for destroy plans (changes phase badge to red) |
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
        run: tf-summarize

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
        run: tf-summarize
```

## Output

Badges use [shields.io](https://shields.io) styling with color-coded change types:
- **Create** (green #28a745) — new resources
- **Modify** (yellow #FFC107) — resource modifications
- **Remove** (red #dc3545) — resource deletions
- **Replace** (red #dc3545) — resource replacements
- **Import** (purple #6f42c1) — imported resources
- **No Changes** (blue #0366d6) — no detected changes
- **Drift Detected** (orange #fd7e14) — drift warnings

Phase badges adapt based on plan type and success:
- **Plan** (blue #007bff) — terraform plan
- **Destroy** (red #dc3545) — destroy plan (when `DESTROY=true`)
- **Apply** (green #28a745) — successful apply
- **Apply** (red #dc3545) — failed apply

### Plan — Create

```
## 📋 Changes found for `plat-ue2-sandbox`

![Terraform](https://img.shields.io/badge/Terraform-Plan-007bff) ![](https://img.shields.io/badge/-Create%20(9)-28a745)

**Plan:** **9** to add

▸ Create (9)          — collapsible resource list
▸ Terraform Plan Output — collapsible full plan diff
```

### Plan — Replace (with destroy warning)

```
## 📋 Changes found for `plat-ue2-sandbox`

![Terraform](https://img.shields.io/badge/Terraform-Plan-007bff) ![](https://img.shields.io/badge/-Create%20(1)-28a745) ![](https://img.shields.io/badge/-Modify%20(1)-FFC107) ![](https://img.shields.io/badge/-Remove%20(2)-dc3545) ![](https://img.shields.io/badge/-Replace%20(1)-dc3545)

> [!CAUTION]
> **Terraform will delete resources!**

**Plan:** **1** to add, **1** to change, **2** to destroy

▸ Update (1)
▸ Destroy (1)
▸ Replace (1)
▸ Terraform Plan Output
```

### Plan — Destroy

```
## 📋 Changes found for `prod`

![Terraform](https://img.shields.io/badge/Terraform-Destroy-dc3545) ![](https://img.shields.io/badge/-Remove%20(5)-dc3545)

> [!CAUTION]
> **Terraform will delete resources!**

**Plan:** **5** to destroy

▸ Destroy (5)
▸ Terraform Plan Output
```

### Apply — Success

```
## ✅ Changes applied successfully for `prod`

![Terraform](https://img.shields.io/badge/Terraform-Apply-28a745) ![](https://img.shields.io/badge/-Create%20(3)-28a745)

**Result:** **3** added

▸ ✅ Created (3)       — each resource listed
▸ Terraform Apply Output
```

### Apply — Partial Failure

```
## ❌ Apply failed for `prod`

![Terraform](https://img.shields.io/badge/Terraform-Apply-dc3545) ![](https://img.shields.io/badge/-Create%20(2)-28a745) ![](https://img.shields.io/badge/-Failed%20(1)-dc3545)

**Result:** **2** added, **1** failed

▸ ✅ Created (2)       — resources that succeeded
▸ ❌ Failed (1)        — open by default, shows resource + error
▸ Terraform Apply Output
```

### No Changes

```
## ✅ No changes found for `dev`

![Terraform](https://img.shields.io/badge/Terraform-Plan-007bff) ![](https://img.shields.io/badge/-No%20Changes-0366d6)

Infrastructure is up-to-date. No changes needed.
```

## Build

### Default build (development version)

```bash
go build -o tf-summarize .
```

Or using the Makefile:

```bash
make build
```

### Build with version injection

To embed a version at build time, use the `-ldflags` flag:

```bash
go build -ldflags "-X main.Version=1.0.0" -o tf-summarize .
```

Or using the Makefile with a VERSION variable:

```bash
make build-versioned VERSION=1.0.0
```

### Display version

```bash
./tf-summarize --version
```

### Display help

```bash
./tf-summarize --help
```

### Release automation

For automated releases with version injection, you can use GitHub Actions with ldflags:

```yaml
- name: Build Release
  run: |
    VERSION=${{ github.ref_name }} make build-versioned
    # Creates binary with version from git tag (e.g., v1.0.0)
```

This allows you to:
- Tag releases in git: `git tag v1.0.0`
- Automatically inject version during CI/CD builds
- Display version info with `--version` flag

## Releases

Releases are automated via GitHub Actions using semantic versioning and [conventional commits](https://www.conventionalcommits.org/).

### How Releases Work

When you push commits to `main`, the workflow automatically:
1. Analyzes commit messages
2. Determines version bump (major, minor, or patch)
3. Builds Linux x86_64 binary with version injected
4. Creates git tag and GitHub release
5. Generates changelog from commits

### Version Bumping

| Commit Type | Version | Example |
|-------------|---------|---------|
| `feat:` | Minor | `feat: add new output format` |
| `fix:` | Patch | `fix: handle edge case` |
| `perf:` | Patch | `perf: optimize parser` |
| `refactor:` | Patch | `refactor: simplify logic` |
| `feat!:` or `BREAKING CHANGE` | Major | `feat!: restructure API` |
| `docs:`, `style:`, `test:`, `chore:` | No release | Internal changes |

### Making a Release

Just push commits with conventional format:

```bash
# Patch release (1.0.0 → 1.0.1)
git commit -m "fix: handle edge case in parser"
git push origin main

# Minor release (1.0.0 → 1.1.0)
git commit -m "feat: add new output format"
git push origin main

# Major release (1.0.0 → 2.0.0)
git commit -m "feat!: restructure API"
git push origin main
```

### Install Releases

Binaries are available for Linux x86_64 at: https://github.com/jomakori/TF_summarize/releases

```bash
# Install and run
go install github.com/jomakori/TF_summarize@latest
tf-summarize --version
```

## Test

```bash
go test ./...
```
