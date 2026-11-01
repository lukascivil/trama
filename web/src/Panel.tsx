import { useState } from "react";
import { edgeGroup, trusted, type Bundle } from "./graph";
import { releaseText, isLink } from "./format";
import { T } from "./i18n";
import type { Selection } from "./marks";
import type { DataEntry, Edge, Hit, Link, Model, Repo } from "./types";
import { Chip, Evidence, EvidenceLink, FilterBox, matches, RepoButton, Section } from "./ui";

export type TabKey = "affected" | "provides" | "consumes" | "tables" | "run" | "changes" | "data";

export type FocusControls = {
  active: boolean;
  levels: number;
  onLevels: (n: number) => void;
  onFocus: (repo: string | null, levels: number) => void;
};

type Select = { onSelectRepo: (name: string) => void };

type Props = Select & {
  model: Model;
  bundles: Bundle[];
  sel: Selection;
  tab: TabKey;
  onTab: (tab: TabKey) => void;
  focus: FocusControls;
};

const statusBadge = (e: Edge) => (
  <span className={trusted(e) ? "badge" : "badge bad"}>{T.status[e.status]}</span>
);

// ---- the repository -------------------------------------------------------

function HitItem({ h, kind, onSelectRepo }: Select & { h: Hit; kind: "direct" | "indirect" | "possible" }) {
  return (
    <li>
      <div>
        <span className={`badge ${kind}`}>{T[kind]}</span> <RepoButton name={h.repo} onSelect={onSelectRepo} />
        {h.id && (
          <>
            {" · "}
            <code>{h.id}</code>
          </>
        )}
      </div>
      {kind === "indirect" && <div className="muted">{T.via(h.path ?? [])}</div>}
      {kind === "possible" && <div className="muted">{T.reason[h.reason ?? ""] ?? h.reason}</div>}
      <Evidence list={h.evidence} />
    </li>
  );
}

function AffectedTab({ repo, onSelectRepo }: Select & { repo: Repo }) {
  const [q, setQ] = useState("");
  const a = repo.affected;
  const all = a.direct.length + a.indirect.length + a.possible.length;
  const keep = (h: Hit) => matches(q, h.repo, h.id, h.kind);
  const groups = [
    { kind: "direct" as const, hits: a.direct.filter(keep) },
    { kind: "indirect" as const, hits: a.indirect.filter(keep) },
    { kind: "possible" as const, hits: a.possible.filter(keep) },
  ];
  if (all === 0) return <p className="muted pad">{T.nobody}</p>;
  return (
    <>
      <p className="muted hint-line">{T.affectedTitle}</p>
      <FilterBox value={q} onChange={setQ} />
      {groups.every((g) => g.hits.length === 0) && <p className="muted">{T.noneMatch}</p>}
      {groups
        .filter((g) => g.hits.length > 0)
        .map((g) => (
          <Section key={g.kind} title={T[g.kind]} count={g.hits.length}>
            <ul>
              {g.hits.map((h, i) => (
                <HitItem key={i} h={h} kind={g.kind} onSelectRepo={onSelectRepo} />
              ))}
            </ul>
          </Section>
        ))}
    </>
  );
}

function ProvidesTab({ repo, model }: { repo: Repo; model: Model }) {
  const [q, setQ] = useState("");
  const fieldsOf = (id: string) => (model.data ?? []).filter((d) => d.repo === repo.name && d.layer !== "table" && d.id === id);
  const list = (repo.provides ?? []).filter((p) => matches(q, p.id, p.kind, ...fieldsOf(p.id).map((d) => d.path)));
  return (
    <>
      <FilterBox value={q} onChange={setQ} />
      {list.length === 0 && <p className="muted">{T.noneMatch}</p>}
      <ul>
        {list.map((p) => {
          const fields = fieldsOf(p.id);
          return (
            <li key={`${p.kind}:${p.id}`}>
              <code>{p.id}</code>
              <span className="muted"> · {p.kind}</span>
              <Evidence list={p.evidence} />
              {fields.length > 0 && (
                <details open={q.trim() !== "" || undefined}>
                  <summary>{T.dataHits(fields.length)}</summary>
                  <ul className="fields">
                    {fields.map((d, i) => (
                      <li key={i}>
                        <span className="muted">{d.layer === "request" ? T.inRequest : T.inResponse}</span> <code>{d.path}</code>
                        {d.type && <span className="muted"> · {d.type}</span>}
                      </li>
                    ))}
                  </ul>
                </details>
              )}
            </li>
          );
        })}
      </ul>
    </>
  );
}

// What the provider says it accepts and returns at this id: the contract the
// consumer is bound to.
function Contract({ fields }: { fields: DataEntry[] }) {
  if (fields.length === 0) return null;
  return (
    <ul className="fields">
      {fields.map((d, i) => (
        <li key={i}>
          <span className="muted">{d.layer === "request" ? T.inRequest : T.inResponse}</span> <code>{d.path}</code>
          {d.type && <span className="muted"> · {d.type}</span>}
        </li>
      ))}
    </ul>
  );
}

// An endpoint shows only its id and status; "see more" opens the files that
// prove it and the contract.
function ConsumedItem({ e, fields, open, onToggle }: { e: Edge; fields: DataEntry[]; open: boolean; onToggle: () => void }) {
  const hasMore = (e.evidence ?? []).length > 0 || fields.length > 0;
  return (
    <li>
      <code>{e.id}</code> {statusBadge(e)}
      {hasMore && (
        <>
          {" "}
          <button type="button" className="more" aria-expanded={open} onClick={onToggle}>
            {open ? T.seeLess : T.seeMore}
          </button>
        </>
      )}
      {open && (
        <>
          <Evidence list={e.evidence} />
          {fields.length > 0 && (
            <>
              <div className="muted contract-title">{T.contract}</div>
              <Contract fields={fields} />
            </>
          )}
        </>
      )}
    </li>
  );
}

function ConsumesTab({ edges, model, onSelectRepo }: Select & { edges: Edge[]; model: Model }) {
  const [q, setQ] = useState("");
  const [closed, setClosed] = useState<Set<string>>(new Set());
  const [opened, setOpened] = useState<Set<string>>(new Set());
  const list = edges.filter((e) => matches(q, e.id, e.provider, e.kind));
  const byProvider = new Map<string, Edge[]>();
  list.forEach((e) => {
    const key = e.provider ?? "?";
    byProvider.set(key, [...(byProvider.get(key) ?? []), e]);
  });
  const itemKey = (e: Edge) => `${e.provider ?? "?"}\u0000${e.kind ?? ""}\u0000${e.id}`;
  const fieldsOf = (e: Edge) =>
    e.provider ? (model.data ?? []).filter((d) => d.repo === e.provider && d.layer !== "table" && d.id === e.id) : [];
  const flip = (set: Set<string>, key: string) => {
    const next = new Set(set);
    if (!next.delete(key)) next.add(key);
    return next;
  };
  // A search must not hide its own answer behind a closed list.
  const searching = q.trim() !== "";
  const providers = [...byProvider.keys()];
  const allClosed = providers.every((p) => closed.has(p));
  const allOpened = list.every((e) => opened.has(itemKey(e)));
  return (
    <>
      <FilterBox value={q} onChange={setQ} />
      {list.length === 0 && <p className="muted">{T.noneMatch}</p>}
      {list.length > 0 && (
        <div className="list-actions">
          {providers.length > 1 && (
            <button type="button" className="more" onClick={() => setClosed(allClosed ? new Set() : new Set(providers))}>
              {allClosed ? T.expandAll : T.collapseAll}
            </button>
          )}
          <button type="button" className="more" onClick={() => setOpened(allOpened ? new Set() : new Set(list.map(itemKey)))}>
            {allOpened ? T.seeLessAll : T.seeMoreAll}
          </button>
        </div>
      )}
      {[...byProvider].map(([provider, items]) => {
        const open = searching || !closed.has(provider);
        return (
          <Section
            key={provider}
            title={`${T.from} ${provider}`}
            count={items.length}
            open={open}
            onToggle={() => setClosed(flip(closed, provider))}
          >
            {provider !== "?" && <RepoButton name={provider} onSelect={onSelectRepo} />}
            <ul>
              {items.map((e) => (
                <ConsumedItem
                  key={itemKey(e)}
                  e={e}
                  fields={fieldsOf(e)}
                  open={opened.has(itemKey(e))}
                  onToggle={() => setOpened(flip(opened, itemKey(e)))}
                />
              ))}
            </ul>
          </Section>
        );
      })}
    </>
  );
}

function TablesTab({ rows }: { rows: DataEntry[] }) {
  const [q, setQ] = useState("");
  const byTable = new Map<string, { columns: DataEntry[]; evidence: Link[] }>();
  rows.forEach((d) => {
    const name = d.table ?? "?";
    const t = byTable.get(name) ?? { columns: [], evidence: d.evidence };
    t.columns.push(d);
    byTable.set(name, t);
  });
  const list = [...byTable].filter(([name, t]) => matches(q, name, ...t.columns.map((c) => c.path)));
  return (
    <>
      <FilterBox value={q} onChange={setQ} />
      {list.length === 0 && <p className="muted">{T.noneMatch}</p>}
      <ul>
        {list.map(([name, t]) => (
          <li key={name}>
            <code>{name}</code>
            <div className="chips">
              {t.columns.map((c) => (
                <span key={c.path} className="chip" title={c.type}>
                  {c.path}
                </span>
              ))}
            </div>
            <Evidence list={t.evidence} />
          </li>
        ))}
      </ul>
    </>
  );
}

function DataItem({ d, onSelectRepo }: Select & { d: DataEntry }) {
  const label = { response: T.inResponse, request: T.inRequest, table: T.inTable }[d.layer] ?? d.layer;
  return (
    <li>
      <div>
        <RepoButton name={d.repo} onSelect={onSelectRepo} /> <span className="muted">{label}</span>{" "}
        <code>{d.layer === "table" && d.table ? `${d.table}.${d.path}` : d.path}</code>
        {d.type && <span className="muted"> · {d.type}</span>}
        {d.id && (
          <>
            {" · "}
            <code>{d.id}</code>
          </>
        )}
      </div>
      <Evidence list={d.evidence} />
    </li>
  );
}

// The hits of a field or column search, in the order the model gives them:
// the endpoints that already serve the data, then the tables that hold it.
function DataList({ hits, onSelectRepo }: Select & { hits: DataEntry[] }) {
  const [q, setQ] = useState("");
  const list = hits.filter((d) => matches(q, d.repo, d.path, d.table, d.id));
  return (
    <>
      <p className="muted">{T.dataHits(hits.length)}</p>
      {hits.length > 6 && <FilterBox value={q} onChange={setQ} />}
      {list.length === 0 && <p className="muted">{T.noneMatch}</p>}
      <ul>
        {list.map((d, i) => (
          <DataItem key={i} d={d} onSelectRepo={onSelectRepo} />
        ))}
      </ul>
    </>
  );
}

function RepoHeader({ repo, focus }: { repo: Repo; focus: FocusControls }) {
  const release = repo.release ? `${releaseText(repo)} · ${T.commitsAfter(repo.release.commitsAfter)}` : releaseText(repo);
  return (
    <header className="repo-head">
      <h2>{isLink(repo.url) ? <EvidenceLink link={{ text: repo.name, url: repo.url }} /> : repo.name}</h2>
      <div className="repo-sub">
        <Chip role={repo.role} />
        <span className="muted">{release}</span>
      </div>
      {repo.description && (
        <p className="desc" title={repo.description}>
          {repo.description}
        </p>
      )}
      {(repo.stack ?? []).length > 0 && (
        <div className="chips">
          {(repo.stack ?? []).map((s) => (
            <span key={s} className="chip">
              {s}
            </span>
          ))}
        </div>
      )}
      {T.card[repo.card] && <p className="badge warn">{T.card[repo.card]}</p>}
      <div className="controls">
        <button type="button" onClick={() => focus.onFocus(repo.name, focus.levels)}>
          {T.focus}
        </button>
        <select value={focus.levels} onChange={(ev) => focus.onLevels(Number(ev.target.value))}>
          {[1, 2, 3].map((n) => (
            <option key={n} value={n}>
              {T.levels(n)}
            </option>
          ))}
        </select>
        {focus.active && (
          <button type="button" onClick={() => focus.onFocus(null, focus.levels)}>
            {T.showAll}
          </button>
        )}
      </div>
    </header>
  );
}

function RepoPanel({ repo, model, extra, tab, onTab, focus, onSelectRepo }: Select & { repo: Repo; model: Model; extra?: DataEntry[]; tab: TabKey; onTab: (t: TabKey) => void; focus: FocusControls }) {
  const consumed = model.edges.filter((e) => e.consumer === repo.name);
  const owned = (model.data ?? []).filter((d) => d.repo === repo.name && d.layer === "table");
  const a = repo.affected;
  const tabs: { key: TabKey; label: string; count?: number }[] = [
    { key: "affected", label: T.tabs.affected, count: a.direct.length + a.indirect.length + a.possible.length },
    { key: "provides", label: T.tabs.provides, count: (repo.provides ?? []).length },
    { key: "consumes", label: T.tabs.consumes, count: consumed.length },
    { key: "tables", label: T.tabs.tables, count: new Set(owned.map((d) => d.table)).size },
    { key: "run", label: T.tabs.run, count: (repo.run ?? []).length },
    { key: "changes", label: T.tabs.changes, count: (repo.openChanges ?? []).length },
    ...(extra ? [{ key: "data" as const, label: T.data, count: extra.length }] : []),
  ];
  const shown = tabs.filter((t) => t.key === "affected" || (t.count ?? 0) > 0);
  const current = shown.some((t) => t.key === tab) ? tab : "affected";

  return (
    <>
      <RepoHeader repo={repo} focus={focus} />
      <nav className="tabs" role="tablist">
        {shown.map((t) => (
          <button key={t.key} type="button" role="tab" aria-selected={t.key === current} className={t.key === current ? "on" : undefined} onClick={() => onTab(t.key)}>
            {t.label}
            {t.count !== undefined && <span className="count">{t.count}</span>}
          </button>
        ))}
      </nav>
      <div className="tab-body" key={`${repo.name}:${current}`}>
        {current === "affected" && <AffectedTab repo={repo} onSelectRepo={onSelectRepo} />}
        {current === "provides" && <ProvidesTab repo={repo} model={model} />}
        {current === "consumes" && <ConsumesTab edges={consumed} model={model} onSelectRepo={onSelectRepo} />}
        {current === "tables" && <TablesTab rows={owned} />}
        {current === "run" && (
          <ul>
            {(repo.run ?? []).map((c) => (
              <li key={c.kind}>
                <strong>{c.kind}</strong> <code>{c.cmd}</code>
                {c.port ? <span className="muted"> · {T.port(c.port)}</span> : null}
              </li>
            ))}
          </ul>
        )}
        {current === "changes" && (
          <ul>
            {(repo.openChanges ?? []).map((c) => (
              <li key={c.id}>
                <EvidenceLink link={{ text: c.id, url: c.url }} />
                <div>{c.title}</div>
              </li>
            ))}
          </ul>
        )}
        {current === "data" && extra && <DataList hits={extra} onSelectRepo={onSelectRepo} />}
      </div>
    </>
  );
}

// ---- an arrow, an id, a field ----------------------------------------------

function EdgeDetail({ e, onSelectRepo }: Select & { e: Edge }) {
  return (
    <div className="plain">
      <h2>{e.id}</h2>
      <p className="muted">{[e.kind, T.status[e.status]].filter(Boolean).join(" · ")}</p>
      <p>
        <RepoButton name={e.consumer} onSelect={onSelectRepo} /> →{" "}
        {e.provider ? <RepoButton name={e.provider} onSelect={onSelectRepo} /> : <strong>?</strong>}
      </p>
      {e.via && (
        <Section title={T.proof}>
          <p>{e.via}</p>
        </Section>
      )}
      {(e.candidates ?? []).length > 0 && (
        <Section title={T.candidates}>
          <p>{(e.candidates ?? []).join(", ")}</p>
        </Section>
      )}
      {(e.evidence ?? []).length > 0 && (
        <Section title={T.evidence}>
          <Evidence list={e.evidence} />
        </Section>
      )}
      {(e.problems ?? []).length > 0 && (
        <Section title={T.problems}>
          <ul>
            {(e.problems ?? []).map((p, i) => (
              <li key={i} className="badge bad">
                {p}
              </li>
            ))}
          </ul>
        </Section>
      )}
    </div>
  );
}

// An arrow stands for one relation or for a list of them, each line opening
// its own proof.
function BundlePanel({ list, onSelectRepo }: Select & { list: Edge[] }) {
  const [open, setOpen] = useState<Edge | null>(list.length === 1 ? list[0] : null);
  const first = list[0];
  if (open) {
    return (
      <>
        {list.length > 1 && (
          <button type="button" className="back" onClick={() => setOpen(null)}>
            ← {first.consumer} → {first.provider ?? "?"}
          </button>
        )}
        <EdgeDetail e={open} onSelectRepo={onSelectRepo} />
      </>
    );
  }
  return (
    <div className="plain">
      <h2>
        {first.consumer} → {first.provider ?? "?"}
      </h2>
      <p className="muted">{T.together(list.length, T.kind[edgeGroup(first)])}</p>
      <ul>
        {list.map((e, i) => (
          <li key={i} className="rowlink" onClick={() => setOpen(e)}>
            <code>{e.id}</code> {statusBadge(e)}
          </li>
        ))}
      </ul>
    </div>
  );
}

function UsesPanel({ id, providers, users, model, onSelectRepo }: Select & { id: string; providers: string[]; users: number[]; model: Model }) {
  return (
    <div className="plain">
      <h2>{id}</h2>
      <Section title={T.provider}>
        {providers.length ? (
          providers.map((p) => <RepoButton key={p} name={p} onSelect={onSelectRepo} />)
        ) : (
          <p>?</p>
        )}
      </Section>
      <Section title={T.consumers} count={users.length}>
        <ul>
          {users.map((i) => {
            const e = model.edges[i];
            return (
              <li key={i}>
                <RepoButton name={e.consumer} onSelect={onSelectRepo} /> <span className="muted">{T.status[e.status]}</span>
                <Evidence list={e.evidence} />
              </li>
            );
          })}
        </ul>
      </Section>
    </div>
  );
}

export function Panel({ model, bundles, sel, tab, onTab, focus, onSelectRepo }: Props) {
  switch (sel.kind) {
    case "none":
      return <p className="muted plain">{T.hint}</p>;
    case "nomatch":
      return <p className="muted plain">{T.noMatch(sel.term)}</p>;
    case "repo": {
      const repo = model.repos.find((r) => r.name === sel.name);
      if (!repo) return null;
      return <RepoPanel repo={repo} model={model} extra={sel.data} tab={tab} onTab={onTab} focus={focus} onSelectRepo={onSelectRepo} />;
    }
    case "bundle": {
      const b = bundles.find((x) => x.id === sel.id);
      if (!b) return null;
      return <BundlePanel key={b.id} list={b.idxs.map((i) => model.edges[i])} onSelectRepo={onSelectRepo} />;
    }
    case "uses":
      return <UsesPanel id={sel.id} providers={sel.providers} users={sel.users} model={model} onSelectRepo={onSelectRepo} />;
    case "data":
      return (
        <div className="plain">
          <h2>{sel.term}</h2>
          <Section title={T.data}>
            <DataList hits={sel.hits} onSelectRepo={onSelectRepo} />
          </Section>
        </div>
      );
  }
}
