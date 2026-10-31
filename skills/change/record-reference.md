# Change record reference

One record per change that spans repositories, at `changes/<id>.md`, committed to the workspace's main branch with `trama changes save <id>` (directly, without a pull request; it refuses to push other unpushed workspace commits along with it). `trama changes` keeps the step states up to date from the pull requests; it rewrites only the front matter, in its own format (comments there are not kept), and never touches the text below it.

## Front matter

| Field | Required | Notes |
|---|---|---|
| `id` | yes | `YYYY-MM-DD-<slug>`, the file name without `.md`. The slug is lowercase words joined by `-` (e.g. `rename-message`). |
| `title` | yes | One line, in the user's terms. |
| `status` | yes | `open` until every step is `merged`, then `done`. `trama changes` sets it. |
| `continues` | no | The `id` of an earlier record whose pending step this change completes. That earlier record's pending step then gets the same `branch`, `pr` and `state: open`, so it closes when the pull request merges. |
| `touches` | yes | The ids the change touches (e.g. `["GET /hello"]`), so later changes can find this record. |
| `steps` | yes | One entry per pull request, in merge order, plus the steps left for later. |

Each step:

| Field | Required | Notes |
|---|---|---|
| `order` | yes | Merge order, from 1. |
| `repo` | yes | Repository name from `workspace.yaml`. |
| `action` | yes | What the step does, in one line. |
| `branch` | when it has a PR | `trama/<slug>`. |
| `pr` | when not `pending` | The pull request URL (`https://…`). A `pending` step has none. |
| `state` | yes | `pending` (no pull request yet), `open`, `merged` or `closed`. |

## Sections

- `## Why`: the request and the reason for it.
- `## Impact`: the answer of `/trama:impact`: repositories, `path:line`, breaking or compatible.
- `## Plan`: the steps in order, expand → migrate → contract.
- `## Execution`: branch, pull requests, tests that ran.
- `## Pending`: steps left for later and when they can happen.
- `## Continues`: only with `continues`, a link to the earlier record.

## Example

````markdown
---
id: 2026-11-03-rename-message
title: Rename message to greeting in GET /hello
status: open
touches: ["GET /hello"]
steps:
  - order: 1
    repo: trama-demo-api
    action: Add greeting to GET /hello, keeping message
    branch: trama/rename-message
    pr: https://github.com/lukascivil/trama-demo-api/pull/1
    state: open
  - order: 2
    repo: trama-demo-web
    action: Read greeting instead of message
    branch: trama/rename-message
    pr: https://github.com/lukascivil/trama-demo-web/pull/1
    state: open
  - order: 3
    repo: trama-demo-api
    action: Remove message from GET /hello
    state: pending
---

## Why

The greeting field should be called `greeting`, not `message`.

## Impact

- `trama-demo-web` reads `message` at `src/api/hello.ts:15`: breaking.
- No other repository uses `GET /hello`.

## Plan

1. `trama-demo-api`: return `greeting` next to `message`.
2. `trama-demo-web`: read `greeting`.
3. `trama-demo-api`: remove `message`, after step 2 is merged and deployed.

## Execution

- Branch `trama/rename-message` in both repositories.
- `trama test` passed in both.
- Pull requests: trama-demo-api#1, then trama-demo-web#1.

## Pending

- Step 3, once every consumer reads `greeting`.
````
