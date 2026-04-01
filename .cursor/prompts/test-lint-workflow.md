# test-lint workflow (canonical)

From the repository root, run **`task`** (default pipeline: lint with auto-fix, unit tests, then integration tests).

**Goal:** keep iterating until **`task` exits 0**.

## Task targets (this repo)

**Full check (preferred end state):**

- **`task`** — root default: **`lint`** (namespace default runs **`lint:fix`**), **`task test`** (unit), **`task test:integration`** (conformance + management).

**Lint (when isolating failures):**

- **`task lint:lint`** — `golangci-lint run` without `--fix` (report / verify clean).
- **`task lint:fix`** — `golangci-lint run --fix`.
- **`task lint`** — same as **`lint:default`** (fix pass only); prefer explicit **`lint:lint`** / **`lint:fix`** for a clear check-then-fix loop.

**Tests (when isolating failures):**

- **`task test`** / **`task test:unit`** — unit tests (`ginkgo`, excludes conformance label).
- **`task test:conformance`**, **`task test:management`**, **`task test:authorization`** — see **`tasks/test.yml`**.

## Workflow

1. Run **`task`**. Capture which step failed (lint vs unit vs integration).
2. **Lint loop:** **`task lint:lint`** → **`task lint:fix`** → **`task lint:lint`** again; repeat **fix** only if needed. Remaining findings: manual edits per **`.cursor/rules/go-standards.mdc`** and **`.cursor/rules/error-handling.mdc`**. **`.golangci.yaml`** is authoritative; do not disable rules in source unless the user asks and the change belongs in that config.
3. **Test failures:** read failure output (package, spec, file:line). Fix production code or tests minimally; follow **`.cursor/rules/testing.mdc`** (Ginkgo/Gomega, `When`/`It`, labels, conformance vs unit layout). Prefer targeted **`task test:*`** while debugging if faster; finish with full **`task`**.
4. Repeat until **`task`** exits **0**.

## Sources of truth

- **`.cursor/rules/testing.mdc`** — suite structure, matchers, packages, labels.
- **`.cursor/rules/go-standards.mdc`**, **`.cursor/rules/error-handling.mdc`** — style and errors for manual lint resolutions.
- **`tasks/test.yml`**, **`tasks/lint.yml`**, root **`Taskfile.yml`** — task names and behavior.

## Output

- Summarize what **`task lint:fix`** or manual edits changed (files and nature of fix).
- Note any remaining risk or follow-up if something could not be fully resolved.
- Confirm final **`task`** succeeds, or state the blocking error and next step.
