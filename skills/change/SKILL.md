---
description: Plan and carry out a change that spans repositories of a trama workspace: record, branches, code, tests and linked pull requests, in a safe order. Use when the user asks to make a change that touches more than one repository.
---

# Change across repositories

Arguments: the change, in the user's words ($ARGUMENTS). Work from the workspace root.

Never push to a repository's main branch, never merge a pull request, never stash, reset or discard anyone's changes, and never skip a repository's tests.

## 1. Impact and plan

1. Look in `changes/` for records whose `touches` include an `id` this change touches and whose steps are not all `merged` (`trama status` lists open changes). If one exists, tell the user and offer to continue it: the new record then sets `continues` to its `id`.
2. Follow the `/trama:impact` skill for the change.
3. Write `changes/<date>-<slug>.md` following `${CLAUDE_PLUGIN_ROOT}/skills/change/record-reference.md`: today's date, a short slug, `touches`, one step per repository and order, expand → migrate → contract. Every step starts as `state: pending`, without `branch` or `pr`; section 3 fills them in for the steps you do now. Write the record in the user's language.
4. Show the plan to the user and wait for their approval. Then run `trama changes save <id>`, which commits the record to the workspace's main branch and prints its link (`record: <url>`). When it prints no link (the workspace is not on GitHub), refer to the record by its path in the workspace repository. If it fails (another branch, a rejected push), stop and tell the user.

## 2. Branches, code and tests

For each step you do now, in order:

1. Run `trama branch <repo> trama/<slug>`. If it refuses (local changes, existing branch), stop and tell the user.
2. Read the repository's `AGENTS.md` or `CLAUDE.md` and follow its rules. Make the step's change and update its tests and docs. Do not add fallbacks for the old form in consumers: the merge order is what keeps the change safe.
3. Run `trama test <repo>`. If it fails, fix it within the step's scope; if you cannot, stop and tell the user.
4. Commit in the repository, following its commit message rules, or the style of its `git log` when it has none, in the repository's language.

## 3. Pull requests

1. For each step, write a body file outside the workspace (a temporary folder, so the workspace stays clean) and run `trama pr <repo> --title "<title>" --body-file <file>`. It prints the pull request URL. Open every step after the first with `--draft` when it must wait for an earlier one to be merged and deployed. The first bodies can be short: step 2 rewrites them.
2. When every pull request exists, rewrite each body with `gh pr edit <url> --body-file <file>` so that all of them carry:
   - one line on the change and a link to the record;
   - every pull request of the change, in merge order, marking this one;
   - the pending steps left for later.
3. In the record, set each step's `branch`, `pr` and `state: open`, fill the Execution section, and run `trama changes save <id>`.
4. Run `trama branch <repo> --main` for every repository you changed. The `trama/<slug>` branches stay.

## 4. Report

Tell the user: the record link, each pull request in merge order, the tests that ran, and the pending steps. Remind them that after the merges, `trama sync`, `trama changes` and `/trama:map --all` bring the workspace up to date.
