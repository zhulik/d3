# Create GitHub issue from chat (canonical)

Use this workflow when filing work that came out of the **current Cursor chat**: the issue should reflect what was discussed, decided, and what remains to do—not a generic project overview.

## Principles

- **Ground the issue in the conversation**: goals, constraints, decisions, file paths, errors, and follow-ups actually mentioned in the thread.
- **Title**: one line, specific, readable on a board (like a good commit subject: clear scope, not vague “Fix bug” or “Improve X”).
- **Body**: Markdown, short sections below. Skip empty sections; do not pad with boilerplate.
- **No secrets**: never paste tokens, passwords, or private URLs from the chat into the issue.

## Body structure (use what applies)

Use these headings when they add value:

1. **`## Summary`** — What the issue is for in one short paragraph.
2. **`## Background`** — Relevant context from the chat (problem statement, reproduction, stack traces trimmed to essentials).
3. **`## Proposed direction`** — If the chat converged on an approach, state it; if not, say options briefly or leave this section out.
4. **`## Acceptance criteria`** — Checklist or bullet list of “done when…” items when the work is definable.
5. **`## Open questions`** — Uncertainties or decisions still needed.

If the chat is only a small fix request, a **`## Summary`** plus bullets may be enough.

## GitHub CLI

- Run from the **repository root** so `gh` targets this repo.
- Require **`gh`** installed and authenticated: **`gh auth status`**. If not logged in, tell the user to run **`gh auth login`** and stop—do not invent issue URLs.
- **Create the issue**: write the body to a **temporary file**, then:

  ```bash
  gh issue create --title "<title>" --body-file <path-to-temp-file>
  ```

- **Print the issue URL** after creation (e.g. from **`gh`** stdout or **`gh issue view <n> --json url -q .url`** if needed).

### Optional flags

- Add **`--label`** only when the user asked for labels **or** when you can infer standard repo labels from existing issues (**`gh label list`**) without guessing wrong.
- Do not assign milestones or users unless the user asked.

## Default vs draft-only

- **Default**: run **`gh issue create`** after showing the user the **title** and **body** (so they can spot mistakes).
- **Draft only**: if the user asked not to create on GitHub (e.g. “text only”, “don’t run gh”), print title and full body Markdown and skip **`gh`**.

## Nothing to file

If the chat has no concrete work item (pure Q&A with no follow-up), say so and do not open an empty issue.
