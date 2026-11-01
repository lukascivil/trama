---
description: "Plan and carry out a change that spans repositories of a trama workspace: record, branches, code, tests and linked pull requests, in a safe order. Use when the user asks to make a change that touches more than one repository."
---

# Change across repositories

Arguments: the change, in the user's words ($ARGUMENTS). Work from the workspace root.

Never push to a repository's main branch, never merge a pull request, never stash, reset or discard anyone's changes, and never skip a repository's tests.

Other skills may shape how you think, plan, test and review, but this skill owns the repositories' mechanics. Work in the clones under `repos/`, never in a git worktree: `trama branch`, `trama test` and `trama pr` only see the clone. Open branches and pull requests only through `trama branch` and `trama pr`, never through another skill's flow for finishing a branch. The plan is the record in `changes/`; do not commit another plan or spec file to the workspace.

## 1. Impact and plan

1. Look in `changes/` for records whose `touches` include an `id` this change touches (for records without `touches`, read their impact section, whatever the language of its heading) and whose steps are not all `merged` (`trama status` lists open changes). If one exists, tell the user and offer to continue it: the new record then sets `continues` to its `id`, and when you open the pull request for the earlier record's pending step, set that step's `branch`, `pr` and `state: open` in the earlier record too and save it with `trama changes save <earlier id>`, so it closes when the pull request merges.
2. Follow the `/trama:impact` skill for the change.
3. Write `changes/<date>-<slug>.md` following `${CLAUDE_PLUGIN_ROOT}/skills/change/record-reference.md`: today's date, a slug, `touches`, one step per repository and order, expand → migrate → contract. The slug also names the `trama/<slug>` branches, so write it in the language the touched repositories use for code and branch names. Every step starts as `state: pending`, without `branch` or `pr`; section 3 fills them in for the steps you do now. In the impact section, name as a risk the tests that cannot catch the break even after the change: mocks of the provider (section 2 moves them to the new form, but they still cannot fail when the provider changes), and tests in repositories the change does not touch. Write the title, each step's `action`, the section headings and their text in the user's language; the front matter keys and the `status` and `state` values stay as the reference gives them.
4. Show the plan to the user and wait for their approval. Then run `trama changes save <id>`, which commits the record to the workspace's main branch and prints its link (`record: <url>`). When it prints no link (the workspace's origin is not on GitHub, or it has none), refer to the record as `changes/<id>.md` followed by the workspace's origin URL (`git remote get-url origin`), or by the path alone when there is no origin. If it fails (another branch, a rejected push), stop and tell the user.

## 2. Branches, code and tests

For each step you do now, in order:

1. Run `trama branch <repo> trama/<slug>`. If it refuses (local changes, the repository on another branch, an existing branch), stop and tell the user.
2. Read the repository's `AGENTS.md` or `CLAUDE.md` and follow its rules. Make the step's change and update its tests and docs. Do not add fallbacks for the old form in consumers: the merge order is what keeps the change safe.
3. If the `/trama:impact` answer named a test in this repository that would not catch the break, fix what this step can: a provider test that decodes with the provider's own types checks the contract itself instead, such as the JSON keys; a consumer test that mocks the provider gets its mock in the new form.
4. Run `trama test <repo>`. If it fails, fix it within the step's scope; if you cannot, stop and tell the user.
5. Commit in the repository, following its commit message rules, or the style of its `git log` when it has none, in the repository's language.

## 3. Pull requests

1. For each step, write a body file outside the workspace (a temporary folder, so the workspace stays clean), in the repository's language and following its pull request template when it has one, and run `trama pr <repo> --title "<the step's commit subject>" --body-file <file>`. It prints the pull request URL. Right away, set that step's `branch`, `pr` and `state: open` in the record and run `trama changes save <id>`, so the record never misses an open pull request even if you stop later. Open every step after the first with `--draft` when it must wait for an earlier one to be merged and deployed. The first bodies can be short: step 2 rewrites them. If setup or tests left generated files that block `trama pr` or `trama branch --main`, tell the user; never delete them.
2. When every pull request exists, rewrite each body with `gh pr edit <url> --body-file <file>` so that all of them carry:
   - one line on the change and a link to the record (without a link, refer to it as section 1 step 4 says);
   - every pull request of the change, in merge order, marking this one;
   - the pending steps left for later.
3. Fill the record's Execution section and run `trama changes save <id>`.
4. Run `trama branch <repo> --main` for every repository you changed. The `trama/<slug>` branches stay.

## 4. Report

Tell the user: the record link (or its path), each pull request in merge order, the tests that ran, and the pending steps. Remind them that after the merges, `trama sync`, `trama changes` and `/trama:map --all` bring the workspace up to date.
