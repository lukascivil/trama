---
description: Create a trama workspace in the current directory. Use when the user asks to start or initialize a trama workspace.
disable-model-invocation: true
---

# Initialize a trama workspace

Create the workspace in the session's current directory. Do not create a GitHub repository and do not commit.

1. Run `trama version`. If it fails, show the error and stop. The usual cause is that `mise` is not installed: point the user to https://mise.jdx.dev.
2. Run `ls -A`. The directory must be empty or contain only `.git`, `.DS_Store` and a `.claude/` folder without `settings.json`. Otherwise stop and tell the user which entries are in the way. Never delete or move them.
3. Run `trama init`. If the user gave a workspace name ($ARGUMENTS), add `--name <name>`.
4. Run `mise trust`, then `mise install`.
5. Tell the user what was created and what comes next:
   - `trama add <url>` for each repository, then `trama sync` to clone them.
   - `trama status` to see the state of every repository.
