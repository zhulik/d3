# amend

Read **`.cursor/prompts/git-conventions-workflow.md`** (sections **Commit messages** and **Creating a commit**).

1. From the repo root, inspect **`git status`**, **`git diff`**, and **`git diff --cached`**. If there is **nothing** to fold into **`HEAD`** (clean working tree and index matches **`HEAD`**), stop and say so—do not run an empty amend.
2. **Do not amend commits that are already pushed.** If **`git rev-parse @{upstream}`** succeeds (branch has an upstream) and **`git rev-list --count @{upstream}..HEAD`** is **0**, then **`HEAD`** is the same commit as the remote tip—**stop** and explain that amending would rewrite published history. (If there is no upstream, this check does not apply.)
3. **Stage** every change that should be included in the amended commit. Prefer **`git add -u`** for updates to **tracked** files (same as **`/commit`**). If **new or untracked** files belong in this commit, stage them too (e.g. **`git add -A`** or explicit paths). Do not amend if the index would still match **`HEAD`**.
4. **Commit message:** either **keep** or **rewrite** the last message.
   - **Keep:** use **`git commit --amend --no-edit`** when the existing message still describes the commit after staging (small fixes, the user said not to change the message, or wording still fits).
   - **Rewrite:** draft a new **subject** and optional **minimal body** per project conventions; **print** them first (**Subject** / **Body**), then **`git commit --amend -m "subject"`** with a second **`-m`** for the body if needed, or **`git commit --amend -F <file>`** for a multi-line body.
5. **Print** the recorded commit description, e.g. **`git log -1 --format=fuller`** (or **`git show -1 --stat`**).

Default behavior is to **run `git commit --amend`** when the guard in step 2 passes. Only skip the amend if the user explicitly asked for text only.
