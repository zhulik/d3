# pr

Read **`.cursor/prompts/git-conventions-workflow.md`** (sections **Pull requests** and **Opening a PR**) and **`.github/pull_request_template.md`**.

1. From the repo root, resolve **base branch**: prefer **`origin/main`** if present, else **`origin/master`**, else **`git symbolic-ref refs/remotes/origin/HEAD`**’s short name.
2. Ensure the current branch is **pushed** ( **`git push -u origin HEAD`** if no upstream or remote is behind what you need for the PR).
3. Build **PR title** (one line) and **full body** Markdown matching the template. Write the body to a **temporary file**.
4. Run **`gh pr create --base <base> --title "<title>" --body-file <path>`** (current branch is used as head). Requires **`gh`** authenticated (**`gh auth status`**).
5. **Print the GitHub PR URL** from **`gh`** output. If it is not obvious, run **`gh pr view --json url -q .url`** for the PR you just created.

Default behavior is to **open the PR with `gh`**. Only output Markdown without **`gh pr create`** if the user explicitly asked for draft text only.
