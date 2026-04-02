# Create GitHub issue from chat (canonical)

Use this workflow when filing work that came out of the **current Cursor chat**: the issue should reflect what was discussed, decided, and what remains to do—not a generic project overview.

## Alignment with `.github/ISSUE_TEMPLATE/`

The repo uses **Bug report**, **Feature request**, and **Other** forms (`bug_report.yml`, `feature_request.yml`, `other.yml`). **Choose one** that fits the chat, then shape title and body to match that template—not a custom outline.

| Template | Title prefix | Default label (`gh issue create --label`) |
|----------|--------------|----------------------------------------|
| Bug report | `[Bug] ` | `bug` |
| Feature request | `[Feature] ` | `enhancement` |
| Other | `[Other] ` | omit unless the user asked for labels |

**Body headings** should mirror the template’s fields (use `##` for each main block). Map chat content into those sections; skip optional blocks only when they add nothing.

- **Bug report**: Summary; Steps to reproduce; Expected behavior; Actual behavior; Environment (d3 version/commit, OS/runtime, storage backend, Redis/Valkey); Additional context.
- **Feature request**: Summary; Problem / motivation; Proposed solution; Alternatives considered; Additional context.
- **Other**: Summary; Details; Additional context.

If something from the chat does not fit a section (e.g. acceptance criteria), fold it into **Additional context** or the closest field rather than inventing new top-level sections.

## Principles

- **Ground the issue in the conversation**: goals, constraints, decisions, file paths, errors, and follow-ups actually mentioned in the thread.
- **Title**: after the required prefix, one line that is specific and readable on a board (clear scope, not vague “Fix bug” or “Improve X”).
- **Body**: Markdown; skip empty sections; do not pad with boilerplate.
- **No secrets**: never paste tokens, passwords, or private URLs from the chat into the issue.

## GitHub CLI

- Run from the **repository root** so `gh` targets this repo.
- Require **`gh`** installed and authenticated: **`gh auth status`**. If not logged in, tell the user to run **`gh auth login`** and stop—do not invent issue URLs.
- **Create the issue**: write the body to a **temporary file**, then:

  ```bash
  gh issue create --title "<title>" --body-file <path-to-temp-file> [--label <name> ...]
  ```

- **Print the issue URL** after creation (e.g. from **`gh`** stdout or **`gh issue view <n> --json url -q .url`** if needed).

### Labels and other flags

- **Bug report** → **`--label bug`**. **Feature request** → **`--label enhancement`**. **Other** → omit **`--label`** unless the user asked for one or you confirm it exists (**`gh label list`**).
- Do not assign milestones or users unless the user asked.

## Default vs draft-only

- **Default**: run **`gh issue create`** after showing the user the **title** and **body** (so they can spot mistakes).
- **Draft only**: if the user asked not to create on GitHub (e.g. “text only”, “don’t run gh”), print title and full body Markdown and skip **`gh`**.

## Nothing to file

If the chat has no concrete work item (pure Q&A with no follow-up), say so and do not open an empty issue.
