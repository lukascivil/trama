import { useEffect, useMemo, useRef, useState } from "react";
import { addRepos, cancel, getRepos } from "./api";
import { filterRepos, setMatching } from "./filter";
import { List } from "./List";
import { lang, T } from "./strings";
import type { AddResult, Repo, ReposResponse } from "./types";

const ROW = 58;

type Phase = "list" | "adding" | "done" | "cancelled";

function RepoRow({ repo, checked, onToggle }: { repo: Repo; checked: boolean; onToggle: () => void }) {
  const date = repo.pushedAt ? new Date(repo.pushedAt).toLocaleDateString(lang) : "";
  const meta = [repo.language, date].filter(Boolean).join(" · ");
  return (
    <label className={repo.blocked ? "row blocked" : "row"}>
      <input type="checkbox" checked={checked || !!repo.blocked} disabled={!!repo.blocked} onChange={onToggle} />
      <span className="row-main">
        <span className="row-name">
          {repo.name}
          {repo.archived ? <span className="tag">archived</span> : null}
        </span>
        <span className="row-desc">{repo.blocked || repo.description}</span>
      </span>
      <span className="row-meta">{meta}</span>
    </label>
  );
}

export function App() {
  const [data, setData] = useState<ReposResponse | null>(null);
  const [loadError, setLoadError] = useState("");
  const [org, setOrg] = useState("");
  const [orgInput, setOrgInput] = useState("");
  const [query, setQuery] = useState("");
  const [showArchived, setShowArchived] = useState(false);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [phase, setPhase] = useState<Phase>("list");
  const [result, setResult] = useState<AddResult | null>(null);
  const [addError, setAddError] = useState("");
  const token = useRef("");

  const load = (name: string) => {
    setData(null);
    setLoadError("");
    getRepos(name)
      .then((d) => {
        token.current = d.token;
        setData(d);
        setOrg(d.org);
        setOrgInput(d.org);
      })
      .catch((e: Error) => setLoadError(e.message));
  };
  useEffect(() => load(""), []);

  const matching = useMemo(() => (data ? filterRepos(data.repos, query, showArchived) : []), [data, query, showArchived]);
  const selectable = matching.filter((r) => !r.blocked).length;

  const changeOrg = (name: string) => {
    setSelected(new Set());
    setQuery("");
    load(name);
  };

  const toggle = (id: string) =>
    setSelected((s) => {
      const next = new Set(s);
      if (!next.delete(id)) next.add(id);
      return next;
    });

  const submit = async () => {
    setPhase("adding");
    setAddError("");
    try {
      const res = await addRepos(token.current, org, [...selected]);
      setResult(res);
      setPhase("done");
    } catch (e) {
      setAddError(T.addError((e as Error).message));
      setPhase("list");
    }
  };

  const abort = async () => {
    try {
      await cancel(token.current);
    } catch {
      // the server may already be gone
    }
    setPhase("cancelled");
  };

  if (phase === "cancelled") return <main className="page"><p className="notice">{T.cancelled}</p></main>;

  if (phase === "done" && result) {
    return (
      <main className="page">
        <h1>{T.title}</h1>
        {result.added.length > 0 ? (
          <>
            <p className="notice ok">{T.added(result.added.length)}</p>
            <ul>{result.added.map((a) => <li key={a.name}>{a.name}</li>)}</ul>
          </>
        ) : null}
        {result.refused.length > 0 ? (
          <>
            <h2>{T.refused}</h2>
            <ul>{result.refused.map((r) => <li key={r.id}><strong>{r.id}</strong>: {r.reason}</li>)}</ul>
          </>
        ) : null}
        {result.added.length > 0 ? (
          <p className="notice">{T.closeTab}</p>
        ) : (
          <button onClick={() => { setPhase("list"); setSelected(new Set()); load(org); }}>{T.back}</button>
        )}
      </main>
    );
  }

  return (
    <main className="page">
      <header className="top">
        <h1>{T.title}</h1>
        {data && data.orgs.length > 0 ? (
          <label className="org">
            {T.org}
            <select value={org} onChange={(e) => changeOrg(e.target.value)}>
              {org === "" ? <option value="" /> : null}
              {data.orgs.map((o) => <option key={o} value={o}>{o}</option>)}
            </select>
          </label>
        ) : data && !data.org ? (
          <form className="org" onSubmit={(e) => { e.preventDefault(); if (orgInput.trim()) changeOrg(orgInput.trim()); }}>
            <input value={orgInput} onChange={(e) => setOrgInput(e.target.value)} placeholder={T.orgPlaceholder} aria-label={T.org} />
            <button type="submit">{T.list}</button>
          </form>
        ) : null}
      </header>

      {loadError ? <p className="notice bad">{T.loadError(loadError)}</p> : null}
      {data && !data.authenticated ? <p className="notice warn">{T.anonymous}</p> : null}
      {data?.error ? <p className="notice bad">{T.loadError(data.error)}</p> : null}
      {addError ? <p className="notice bad">{addError}</p> : null}
      {!data && !loadError ? <p className="notice">{T.loading}</p> : null}

      {data && org ? (
        <>
          <div className="tools">
            <input autoFocus className="filter" type="search" value={query} onChange={(e) => setQuery(e.target.value)} placeholder={T.filter} aria-label={T.filter} />
            <button onClick={() => setSelected((s) => setMatching(s, matching, true))} disabled={selectable === 0}>{T.markMatching(selectable)}</button>
            <button onClick={() => setSelected((s) => setMatching(s, matching, false))} disabled={matching.length === 0}>{T.unmarkMatching}</button>
            <button onClick={() => setShowArchived((v) => !v)}>{showArchived ? T.hideArchived : T.showArchived}</button>
          </div>
          {data.repos.length === 0 && !data.error ? <p className="notice">{T.empty}</p> : null}
          {data.repos.length > 0 && matching.length === 0 ? <p className="notice">{T.noMatch}</p> : null}
          <List items={matching} rowHeight={ROW} itemKey={(r) => r.id} render={(r) => <RepoRow repo={r} checked={selected.has(r.id)} onToggle={() => toggle(r.id)} />} />
        </>
      ) : data && !data.org ? (
        <p className="notice">{T.pickOrg}</p>
      ) : null}

      <footer className="bottom">
        <span>{T.selected(selected.size)}</span>
        <span className="spacer" />
        <button onClick={abort} disabled={phase === "adding"}>{T.cancel}</button>
        <button className="primary" onClick={submit} disabled={selected.size === 0 || phase === "adding"}>
          {phase === "adding" ? T.adding : T.add(selected.size)}
        </button>
      </footer>
    </main>
  );
}
