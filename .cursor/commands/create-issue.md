# create issue

Read **`.cursor/prompts/create-issue-workflow.md`** in this repository.

Issues must match **`.github/ISSUE_TEMPLATE/`**: pick **Bug report**, **Feature request**, or **Other** (see the `*.yml` files there), use that template’s **title prefix** (`[Bug]`, `[Feature]`, or `[Other]`), **labels** when the template defines them (`bug` or `enhancement`), and structure the **body** with the same sections/fields as that form. Blank issues are disabled—do not invent a one-off layout that bypasses these templates.

1. Summarize the **current chat** into a GitHub issue **title** and **body** per that prompt (grounded in the thread; no secrets).
2. **Print** the title and full body Markdown, clearly labeled, before any GitHub action.
3. From the repo root, ensure **`gh auth status`** succeeds, then **`gh issue create`** with **`--body-file`** as described in the workflow. **Print** the new issue URL.
4. If the user asked for **draft text only**, skip **`gh issue create`**.
