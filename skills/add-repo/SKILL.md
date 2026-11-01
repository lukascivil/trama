---
description: Add repositories to the trama workspace, clone them and map them. Use when the user asks to add or include a repository, or to pick repositories from a GitHub organization.
---

# Add repositories

Arguments: a repository URL with optional `--name <name>` and `--branch <branch>`, or an organization name, or nothing ($ARGUMENTS). Work from the workspace root.

## 1. Choose the path

- The argument has a `/` or a `:`, or ends in `.git`: it is a repository. Follow "One repository".
- The argument is a plain word and a directory of that name exists in the current directory: it is a local repository, as it always was. Follow "One repository".
- The argument is a plain word with no such directory: it is an organization. Follow "Pick from an organization".
- There is no argument: follow "Pick from an organization" with no organization; the page asks for it.

## One repository

1. Run `trama add` with the same arguments. If it fails, show the error and stop.
2. Run `trama sync <name>`, using the name `trama add` printed.
3. Map it by following the `/trama:map <name>` skill.

## Pick from an organization

1. Run `trama add --pick [<org>]` **in the background** (it waits for a person, and can take minutes). Tell the user that a page opened in their browser, where they filter, tick the repositories and press Add. If no page opened, give them the URL printed on stderr. Then wait for the command to end.
2. If it fails, show the error and stop. A GitHub error such as an SSO or rate-limit message is theirs to fix; do not retry in a loop.
3. Read its stdout: the names of the repositories that were added, one per line. Lines on stderr that start with `refused` say which ones were not added and why: tell the user.
4. If stdout is empty, tell the user nothing was added and stop.
5. Run `trama sync <name>...` with all the names.
6. Map every added repository, one at a time, by following the `/trama:map <name>` skill. Do not ask first and do not stop at some number: mapping is part of adding, however many the user picked.
