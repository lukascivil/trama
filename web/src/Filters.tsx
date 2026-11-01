import type { ReactNode } from "react";
import { useMemo, useState } from "react";
import { edgeGroup, KINDS, roleOf } from "./graph";
import { T } from "./i18n";
import type { Model } from "./types";
import { matches, FilterBox } from "./ui";

const ROLES = ["api", "web", "app", "worker", "lib", "infra", "other"];

export type FilterState = {
  hiddenRepos: ReadonlySet<string>;
  hiddenKinds: ReadonlySet<string>;
  hiddenRoles: ReadonlySet<string>;
  hideIsolated: boolean;
  onlyChanges: boolean;
};

type Props = {
  model: Model;
  state: FilterState;
  onChange: (next: Partial<FilterState>) => void;
};

const toggled = (set: ReadonlySet<string>, value: string, on: boolean) => {
  const next = new Set(set);
  if (on) next.delete(value);
  else next.add(value);
  return next;
};

function Check({ label, checked, onChange, extra, title }: { label: string; checked: boolean; onChange: (on: boolean) => void; extra?: ReactNode; title?: string }) {
  return (
    <label className="check" title={title}>
      <input type="checkbox" checked={checked} onChange={(ev) => onChange(ev.target.checked)} />
      <span className="check-label">{label}</span>
      {extra}
    </label>
  );
}

// The left drawer: choose which relations, which kinds of repository and
// which repositories are on the canvas.
export function Filters({ model, state, onChange }: Props) {
  const [query, setQuery] = useState("");

  const perKind = useMemo(() => {
    const n = new Map<string, number>();
    model.edges.forEach((e) => n.set(edgeGroup(e), (n.get(edgeGroup(e)) ?? 0) + 1));
    return n;
  }, [model]);

  const perRole = useMemo(() => {
    const n = new Map<string, number>();
    model.repos.forEach((r) => n.set(roleOf(r.role), (n.get(roleOf(r.role)) ?? 0) + 1));
    return n;
  }, [model]);

  const listed = model.repos.filter((r) => matches(query, r.name));
  const setRepos = (names: string[], on: boolean) => {
    const next = new Set(state.hiddenRepos);
    names.forEach((n) => (on ? next.delete(n) : next.add(n)));
    onChange({ hiddenRepos: next });
  };
  const only = (name: string) => onChange({ hiddenRepos: new Set(model.repos.map((r) => r.name).filter((n) => n !== name)) });

  return (
    <aside className="filters">
      <div className="group-head">
        <h2>{T.relations}</h2>
        <span className="bulk">
          <button type="button" onClick={() => onChange({ hiddenKinds: new Set() })}>{T.all}</button>
          <button type="button" onClick={() => onChange({ hiddenKinds: new Set(KINDS) })}>{T.none}</button>
        </span>
      </div>
      {KINDS.map((k) => (
        <Check
          key={k}
          label={T.kind[k]}
          checked={!state.hiddenKinds.has(k)}
          onChange={(on) => onChange({ hiddenKinds: toggled(state.hiddenKinds, k, on) })}
          extra={<span className="count">{perKind.get(k) ?? 0}</span>}
        />
      ))}

      {perRole.size > 1 && (
        <>
          <div className="group-head">
            <h2>{T.types}</h2>
          </div>
          {ROLES.filter((r) => perRole.has(r)).map((r) => (
            <Check
              key={r}
              label={T.role[r] ?? r}
              checked={!state.hiddenRoles.has(r)}
              onChange={(on) => onChange({ hiddenRoles: toggled(state.hiddenRoles, r, on) })}
              extra={<span className="count">{perRole.get(r)}</span>}
            />
          ))}
        </>
      )}

      <div className="group-head">
        <h2>{T.view}</h2>
      </div>
      <Check label={T.hideIsolated} checked={state.hideIsolated} onChange={(on) => onChange({ hideIsolated: on })} />
      <Check label={T.onlyChanges} checked={state.onlyChanges} onChange={(on) => onChange({ onlyChanges: on })} />

      <div className="group-head">
        <h2>{T.repos}</h2>
        <span className="bulk">
          <button type="button" onClick={() => setRepos(listed.map((r) => r.name), true)}>{T.all}</button>
          <button type="button" onClick={() => setRepos(listed.map((r) => r.name), false)}>{T.none}</button>
        </span>
      </div>
      {model.repos.length > 6 && <FilterBox value={query} onChange={setQuery} />}
      {listed.map((r) => (
        <Check
          key={r.name}
          label={r.name}
          checked={!state.hiddenRepos.has(r.name)}
          onChange={(on) => setRepos([r.name], on)}
          extra={
            <button
              type="button"
              className="only"
              title={T.onlyTitle(r.name)}
              onClick={(ev) => {
                ev.preventDefault();
                only(r.name);
              }}
            >
              {T.only}
            </button>
          }
        />
      ))}
    </aside>
  );
}
