# The graph page

The page `trama graph --serve` opens. It is React, React Flow and dagre, built
by Vite into **one script and one stylesheet** in `../internal/view/assets/`,
which the `trama` binary embeds (`go:embed`).

Those two files are committed. Whoever uses or builds trama needs only the
binary or Go; Node is for whoever changes this folder. CI rebuilds the bundle
and fails when the committed one is not what the source produces.

Every relation rule lives in Go (`internal/view`). The page draws the model
`GET /api/workspace` returns (`src/types.ts`) and decides nothing about what
is proven, direct or indirect.

## Working on it

```bash
npm install
```

With a made-up workspace, no trama needed:

```bash
TRAMA_MOCK=1 npm run dev
```

With a real one (the dev server proxies `/api` to it):

```bash
trama graph --serve --no-open --port 7777   # in the workspace
npm run dev                                   # TRAMA_PORT=... if not 7777
```

Before committing a change here:

```bash
npm run typecheck && npm test && npm run build
```

and commit the rebuilt `internal/view/assets/app.js` and `app.css` with it.

## Layout of `src/`

| File | What it is |
|---|---|
| `graph.ts` | arrows (one per pair of repos and kind), filters, focus |
| `marks.ts` | what a selection marks: selected, direct, indirect, possible, faded |
| `search.ts` | what the top box answers |
| `layout.ts` | dagre, left to right or top to bottom, whichever fits the canvas |
| `Canvas.tsx`, `nodes.tsx` | React Flow, the repo card and the `?` node |
| `Filters.tsx`, `Panel.tsx` | the left and right drawers |
| `i18n.ts` | English and Portuguese texts |

The page must stay self-contained: the server sends a CSP of
`default-src 'self'`, and `internal/view/assets_test.go` fails when the bundle
names an address or calls anything but `/api/workspace`.
