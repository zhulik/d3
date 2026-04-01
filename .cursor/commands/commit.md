# commit

Read **`.cursor/prompts/git-conventions-workflow.md`** (sections **Commit messages** and **Creating a commit**).

1. From the repo root, inspect **`git status`**, **`git diff --cached`**, and **`git diff`**. If **nothing is staged** but there are modifications to **tracked** files, run **`git add -u`** so the commit includes them. If there is still nothing to commit, stop and say so—do not run an empty commit.
2. Draft the **subject** and optional **minimal body** per project conventions.
3. **Print** the full message first, labeled clearly (**Subject** / **Body**).
4. Create the commit: prefer **`git commit -m "subject"`** with a second **`-m`** for the body if needed; for multi-line bodies use **`git commit -F <file>`** with a temporary file.
5. **Print** the recorded commit description, e.g. **`git log -1 --format=fuller`** (or **`git show -1 --stat`**).

Default behavior is to **run `git commit`** after printing the message. Only skip the commit if the user explicitly asked for text only.
