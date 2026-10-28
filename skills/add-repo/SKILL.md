---
description: Add a repository to the trama workspace, clone it and map it. Use when the user asks to add or include a repository in the workspace.
---

# Add a repository

Arguments: the repository URL, optionally `--name <name>` and `--branch <branch>` ($ARGUMENTS). Work from the workspace root.

1. Run `trama add` with the same arguments. If it fails, show the error and stop.
2. Run `trama sync <name>`, using the name `trama add` printed.
3. Map it by following the `/trama:map <name>` skill.
