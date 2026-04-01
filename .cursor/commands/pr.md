# pr

Read **`.cursor/prompts/git-conventions-workflow.md`** (sections **Pull requests** and **Opening a PR**) and **`.github/pull_request_template.md`**.

1. From the repo root, resolve **base branch**: prefer **`origin/main`** if present, else **`origin/master`**, else **`git symbolic-ref refs/remotes/origin/HEAD`**’s short name.
2. Ensure the current branch is **pushed** ( **`git push -u origin HEAD`** if no upstream or remote is behind what you need for the PR).
3. Build **PR title** (one line) and **full body** Markdown matching the template. Base the summary on the **current** branch state vs the base branch ( **`git diff <base>...HEAD`**, **`git log <base>..HEAD`** ) so new commits and file changes are included—not stale text from an earlier draft. Write the body to a **temporary file**.
4. Check whether a pull request already exists for the current branch, e.g. **`gh pr view --json number -q .number`** (succeeds when an open PR is linked to **`HEAD`**). Requires **`gh`** authenticated (**`gh auth status`**).
5. **If a PR already exists:** run **`gh pr edit --title "<title>" --body-file <path>`** so the title and body match the template and **recently added** branch changes. Then **print the GitHub PR URL** with **`gh pr view --json url -q .url`** (or from **`gh pr edit`** output if it includes the URL).
6. **If no PR exists:** run **`gh pr create --base <base> --title "<title>" --body-file <path>`** (current branch is used as head). **Print the GitHub PR URL** from **`gh`** output. If it is not obvious, run **`gh pr view --json url -q .url`** for the PR you just created.

Default behavior is to **create or update the PR with `gh`**. Only output Markdown without **`gh pr create`** / **`gh pr edit`** if the user explicitly asked for draft text only.
