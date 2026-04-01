# Git conventions workflow (canonical)

Authoritative template for PRs: **`.github/pull_request_template.md`**.

## Branch naming

### Structure

1. **Optional issue prefix** (when the GitHub issue id is known): `issue-<id>/` then the rest of the name.
2. **Change type** (required after the issue prefix, or at the start if no issue): one of `fix`, `feature`, `doc`, `shore`, `ci`, `update`.
3. **Brief summary** of the change in kebab-case; be specific enough to identify the work at a glance.

Valid patterns:

- `<type>-<summary>`
- `issue-<n>/<type>-<summary>`

### Good

- `fix-timeout-in-redis-client`
- `issue-234/feature-implement-get-versions-method`
- `issue-234/doc-add-compatibility-md`
- `shore-refactor-folder-backend`
- `issue-456/update-vulnerable-foobar`

### Bad

- `fix`, `shore`, `doc` (no summary)
- `fix-issue-14` (issue id belongs in optional prefix, not glued to `fix`)
- `issue-14/fix` (summary too vague)
- `shore/issue123` (wrong separator or missing type-summary shape)
- `shore/refactor-foobar` (use hyphen after type: `shore-refactor-foobar`)

## Commit messages

- **Short title** is the priority; avoid body text when the title is enough.
- **Imperative mood** (e.g. “Add retry”, “Fix nil dereference”, not “Added” / “Fixes”).
- **Not verbose**: no essays, no redundant “this commit” boilerplate.

### Creating a commit (assistant steps)

1. Inspect **`git status`** and **`git diff`** (prefer **staged** changes; include unstaged if the user is committing everything).
2. Draft one **title line** (and optional **minimal body** only when the title is insufficient).
3. Match conventions in **Commit messages** above.
4. **Do not** run **`git commit`** unless the user explicitly asks you to execute it; default is to output text they can paste or use.

## Pull requests

1. **Follow** **`.github/pull_request_template.md`** exactly: fill Summary, Type of change, Breaking changes, Testing, Compatibility / docs as appropriate. Do not omit sections the template expects unless the user says to strip them.
2. **Summarize what the branch actually changes** and **why**; stay concise and highlight what matters for reviewers.
3. If the branch has **many clear, sequential commits**, suggest reviewing **commit-by-commit** in addition to the file diff.
4. If work maps to a GitHub issue, note that the PR **closes** or **fixes** it (e.g. `Fixes #123` / `Closes #456`) in Summary per the template’s guidance.

### Opening a PR (assistant steps)

1. Read **`.github/pull_request_template.md`** and reproduce its **headings and checklists** in your output.
2. Infer **type of change** from the branch name prefix (`fix-`, `feature-`, `doc-`, etc.) when possible; ask if unclear.
3. Summarize **what** changed and **why** in **Summary**; keep **Testing** and **Compatibility / docs** accurate and checked appropriately.
4. Output **Markdown ready to paste** into GitHub (e.g. new PR description field).
5. **Do not** open the PR in the browser or call `gh pr create` unless the user explicitly asks you to run those commands.

## Output expectations

- **Branches:** one primary name plus short alternatives only if useful.
- **Commits:** title line(s) and optional minimal body only when necessary.
- **PRs:** full Markdown body aligned with the template sections.
