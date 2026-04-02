# pr

Read **`.cursor/prompts/git-conventions-workflow.md`** (sections **Pull requests** and **Opening a PR**) and **`.github/pull_request_template.md`**.

1. From the repo root, resolve **base branch**: prefer **`origin/main`** if present, else **`origin/master`**, else **`git symbolic-ref refs/remotes/origin/HEAD`**’s short name.
2. **Push to GitHub before any `gh pr` step:** publish the current branch so **`origin`** matches local **`HEAD`** **before** **`gh pr view`**, **`gh pr edit`**, or **`gh pr create`**. Use **`git push -u origin HEAD`** when there is no upstream; otherwise **`git push`**. Do not open or update a PR until this succeeds (resolve rejections—e.g. non-fast-forward—with the user if needed).
3. Build **PR title** (one line) and **full body** Markdown matching the template. Base the summary on the **current** branch state vs the base branch ( **`git diff <base>...HEAD`**, **`git log <base>..HEAD`** ) so new commits and file changes are included—not stale text from an earlier draft. Write the body to a **temporary file**.
4. Check whether a pull request already exists for the current branch, e.g. **`gh pr view --json number -q .number`** (succeeds when an open PR is linked to **`HEAD`**). Requires **`gh`** authenticated (**`gh auth status`**).
5. **If a PR already exists:** run **`gh pr edit --title "<title>" --body-file <path>`** so the title and body match the template and **recently added** branch changes. Then **print the GitHub PR URL** with **`gh pr view --json url -q .url`** (or from **`gh pr edit`** output if it includes the URL).
6. **If no PR exists:** run **`gh pr create --base <base> --title "<title>" --body-file <path>`** (current branch is used as head). **Print the GitHub PR URL** from **`gh`** output. If it is not obvious, run **`gh pr view --json url -q .url`** for the PR you just created.

Default behavior is to **push, then create or update the PR with `gh`**. Only output Markdown without **`git push`** / **`gh pr create`** / **`gh pr edit`** if the user explicitly asked for draft text only.
