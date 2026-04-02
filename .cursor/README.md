# Cursor layout (d3)

This directory holds **team Cursor config**: rules, agents, slash-command stubs, and long workflows. **If you add, remove, rename, or materially change behavior or scope of anything here, update this file in the same change** (see `.cursor/rules/cursor-readme-sync.mdc`).

## Rules (`.cursor/rules/*.mdc`)

| File | Role |
|------|------|
| `go-standards.mdc` | Core Go conventions (context, errors, logging, `filepath`, `lo`, generics, `task` commands). **Always apply.** |
| `error-handling.mdc` | Sentinel errors, wrapping, `errors.Is`, HTTP mapping via Echo middleware. **Always apply.** |
| `service-architecture.mdc` | `pal` services, `Provide()`, lifecycle, DI. **Always apply.** |
| `api-handlers.mdc` | Echo handler patterns. **Applies to** `internal/server/**/*.go`. |
| `testing.mdc` | Ginkgo/Gomega structure and labels. **Applies to** `**/*_test.go`. |
| `git-workflow.mdc` | Branch names, commits, PRs; points at `git-conventions-workflow.md` and `.github/pull_request_template.md`. On demand (not always on). |
| `cursor-readme-sync.mdc` | When editing under `.cursor/**`, keep this README and cross-references (agents, commands, prompts, rules) aligned. **Applies to** `.cursor/**`. |

## Agents (`.cursor/agents/*.md`)

Used from the Agent picker / delegated tasks. Prefer the specialist that matches the job.

| Agent | Use when |
|-------|----------|
| `issue-orchestrator` | Implement what a **GitHub issue** (or stated scope) describes: implement → **`go-reviewer`** → **`pattern-refine`** → **`security-review`** (fix loop) → **`technical-writer`**; uses **Task**/delegation when the session supports it, otherwise runs each phase per **`.cursor/agents/*.md`**. |
| `coder` | Implement or fix Go code; after substantive edits run **`task`** per `prompts/test-lint-workflow.md` until green. Prefer small, logical commits; keep feature/fix and its tests in one commit (see `agents/coder.md`). |
| `go-reviewer` | Convention and architecture review (no Git hygiene). |
| `pattern-refine` | Duplication, abstraction level, and complexity after implementation; DRY and simpler structure without replacing **go-reviewer** or **security-review**. |
| `security-review` | Threat review for storage paths, locking, checksums, HTTP surface, secrets. |
| `technical-writer` | User/dev docs; AWS-aligned text via AWS Documentation MCP when relevant. |

## Slash commands (`.cursor/commands/*.md`)

Short instructions the editor injects; they usually point at a **prompt** for the full steps.

| Command | Does |
|---------|------|
| `test-lint.md` | Run workflow in `prompts/test-lint-workflow.md` until **`task`** exits 0. |
| `commit.md` | Git commit flow via `prompts/git-conventions-workflow.md`. |
| `amend.md` | `git commit --amend` flow (guards for pushed branches). |
| `pr.md` | Push branch, create or update GitHub PR with `gh`, body from template + conventions prompt. |
| `create-issue.md` | Summarize the current chat and create a GitHub issue with `gh` via `prompts/create-issue-workflow.md`; body/title follow `.github/ISSUE_TEMPLATE/`. |

## Prompts (`.cursor/prompts/*.md`)

Longer canonical workflows; **single source of truth** for multi-step procedures.

| Prompt | Purpose |
|--------|---------|
| `test-lint-workflow.md` | Lint/test loops, `task` targets, pointers to `testing.mdc` and Taskfiles. |
| `git-conventions-workflow.md` | Branches, commits, PRs; aligns with `git-workflow.mdc` and the PR template. |
| `create-issue-workflow.md` | Turn the current chat into a GitHub issue title/body matching `.github/ISSUE_TEMPLATE/`; `gh issue create` with guards (auth, draft-only). |

## Repo root

- **`.cursorignore`** — patterns excluded from Cursor indexing/context (local data, caches, build artifacts, `bin/`, `tmp/`, etc.). If you change what belongs out of index, adjust this file; no need to duplicate every entry in this README.

## Related repo files (outside `.cursor/`)

- **`Taskfile.yml`**, **`tasks/*.yml`** — build, lint, test entrypoints referenced by agents and prompts.
- **`.github/pull_request_template.md`** — required PR shape for `/pr` and conventions.
- **`.github/ISSUE_TEMPLATE/*.yml`** — bug / feature / other issue forms; `/create-issue` and `create-issue-workflow.md` align with these.
- **`.golangci.yaml`** — linter config; do not “fix” lint by disabling rules in source unless agreed.
