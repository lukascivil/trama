import { ReactFlowProvider } from "@xyflow/react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Canvas, type Reveal } from "./Canvas";
import { Filters, type FilterState } from "./Filters";
import { buildBundles, neighborhood, repoID, trusted, visibleGraph } from "./graph";
import { T } from "./i18n";
import { computeMarks, type Selection } from "./marks";
import { Panel, type TabKey } from "./Panel";
import { search } from "./search";
import type { Model } from "./types";

function Banner({ title, lines }: { title: string; lines: string[] }) {
  const [open, setOpen] = useState(false);
  return (
    <div className="banner">
      <button type="button" onClick={() => setOpen(!open)}>
        {title}
      </button>
      {open && lines.length > 0 && (
        <ul>
          {lines.map((l, i) => (
            <li key={i}>{l}</li>
          ))}
        </ul>
      )}
    </div>
  );
}

function Top({ model, onSearch }: { model?: Model; onSearch?: (q: string) => void }) {
  const timer = useRef<number>(0);
  const proven = model ? model.edges.filter(trusted).length : 0;
  return (
    <header className="top">
      <span className="brand">trama</span>
      {model && <strong>{model.workspace}</strong>}
      {model && <span className="muted">{T.counts(model.repos.length, proven, model.edges.length - proven)}</span>}
      <span className="spacer" />
      {onSearch && (
        <input
          className="search"
          type="search"
          autoComplete="off"
          spellCheck={false}
          placeholder={T.search}
          onChange={(ev) => {
            const q = ev.target.value;
            window.clearTimeout(timer.current);
            timer.current = window.setTimeout(() => onSearch(q), 200);
          }}
        />
      )}
    </header>
  );
}

function Workspace({ model }: { model: Model }) {
  const built = useMemo(() => buildBundles(model), [model]);

  const [filters, setFilters] = useState<FilterState>({
    hiddenRepos: new Set(),
    hiddenKinds: new Set(),
    hiddenRoles: new Set(),
    hideIsolated: false,
    onlyChanges: false,
  });
  const [focus, setFocus] = useState<{ repo: string; levels: number } | null>(null);
  const [focusLevels, setFocusLevels] = useState(1);
  const [sel, setSel] = useState<Selection>({ kind: "none" });
  const [tab, setTab] = useState<TabKey>("affected");
  const [reveal, setReveal] = useState<Reveal | null>(null);

  const focusIds = useMemo(() => (focus ? neighborhood(built.bundles, repoID(focus.repo), focus.levels) : null), [built, focus]);
  const visible = useMemo(() => visibleGraph(model, built, { ...filters, focus: focusIds }), [model, built, filters, focusIds]);
  const marks = useMemo(() => computeMarks(model, built.bundles, sel), [model, built, sel]);

  const select = useCallback((name: string, show = false) => {
    setSel({ kind: "repo", name });
    if (show) setReveal((r) => ({ name, n: (r?.n ?? 0) + 1 }));
  }, []);
  const clear = useCallback(() => setSel({ kind: "none" }), []);

  const runSearch = useCallback(
    (q: string) => {
      const r = search(model, q);
      setSel(r.sel);
      if (r.sel.kind === "repo" && r.sel.data) setTab("data");
      const name = r.reveal;
      if (name) setReveal((old) => ({ name, n: (old?.n ?? 0) + 1 }));
    },
    [model],
  );

  useEffect(() => {
    const onKey = (ev: KeyboardEvent) => {
      if (ev.key === "Escape") clear();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [clear]);

  const banner = model.warnings.length > 0 ? <Banner title={T.warnings(model.warnings.length)} lines={model.warnings} /> : null;

  return (
    <>
      <Top model={model} onSearch={runSearch} />
      {banner}
      <main>
        <Filters model={model} state={filters} onChange={(next) => setFilters((f) => ({ ...f, ...next }))} />
        <section className="graph-area">
          <ReactFlowProvider>
            <Canvas
              model={model}
              visible={visible}
              marks={marks}
              reveal={reveal}
              onSelectRepo={select}
              onSelectBundle={(id) => setSel({ kind: "bundle", id })}
              onClear={clear}
            />
          </ReactFlowProvider>
        </section>
        <aside className="panel">
          <Panel
            model={model}
            bundles={built.bundles}
            sel={sel}
            tab={tab}
            onTab={setTab}
            onSelectRepo={(name) => select(name, true)}
            focus={{
              active: focus !== null,
              levels: focusLevels,
              onLevels: setFocusLevels,
              onFocus: (repo, levels) => setFocus(repo ? { repo, levels } : null),
            }}
          />
        </aside>
      </main>
    </>
  );
}

export function App() {
  const [state, setState] = useState<{ model?: Model; error?: string }>({});

  useEffect(() => {
    fetch("/api/workspace", { cache: "no-store" })
      .then(async (res) => {
        const body = await res.json();
        if (!res.ok) throw new Error(body.error || res.statusText);
        return body as Model;
      })
      .then((model) => setState({ model }))
      .catch((err: unknown) => setState({ error: err instanceof Error ? err.message : String(err) }));
  }, []);

  if (state.model) return <Workspace model={state.model} />;
  return (
    <>
      <Top />
      {state.error ? <Banner title={T.loadError(state.error)} lines={[]} /> : <p className="muted plain">{T.loading}</p>}
    </>
  );
}
