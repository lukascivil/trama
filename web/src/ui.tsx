import type { ReactNode } from "react";
import { isLink } from "./format";
import { T } from "./i18n";
import type { Link } from "./types";

// Text from the cards, the records and git always goes in as text: React
// escapes it, and nothing here sets HTML.

export function EvidenceLink({ link }: { link: Link }) {
  if (isLink(link.url)) {
    return (
      <a href={link.url} target="_blank" rel="noopener noreferrer">
        {link.text}
      </a>
    );
  }
  return <code>{link.text}</code>;
}

export function Evidence({ list }: { list?: Link[] }) {
  if (!list || list.length === 0) return null;
  return (
    <div className="evidence">
      {list.map((l, i) => (
        <EvidenceLink key={i} link={l} />
      ))}
    </div>
  );
}

export function FilterBox({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  return (
    <input
      className="filterbox"
      type="search"
      value={value}
      placeholder={T.filter}
      autoComplete="off"
      spellCheck={false}
      onChange={(ev) => onChange(ev.target.value)}
    />
  );
}

// With `onToggle` the heading becomes a button that closes and opens the
// section, so a long list can be folded out of the way.
export function Section({
  title,
  count,
  open = true,
  onToggle,
  children,
}: {
  title: string;
  count?: number;
  open?: boolean;
  onToggle?: () => void;
  children: ReactNode;
}) {
  const label = (
    <>
      {title}
      {count !== undefined && <span className="count">{count}</span>}
    </>
  );
  return (
    <section className="section">
      <h3>
        {onToggle ? (
          <button type="button" className="fold" aria-expanded={open} title={open ? T.collapse : T.expand} onClick={onToggle}>
            <span className="chevron" aria-hidden="true">
              {open ? "▾" : "▸"}
            </span>
            {label}
          </button>
        ) : (
          label
        )}
      </h3>
      {open && children}
    </section>
  );
}

export function Chip({ role }: { role?: string }) {
  if (!role) return null;
  return <span className={`role role-${role}`}>{T.role[role] ?? role}</span>;
}

// RepoButton is a repository name that selects it when clicked.
export function RepoButton({ name, onSelect }: { name: string; onSelect: (name: string) => void }) {
  return (
    <button type="button" className="repolink" title={T.select} onClick={() => onSelect(name)}>
      {name}
    </button>
  );
}

export const matches = (q: string, ...texts: (string | undefined)[]) => {
  const lower = q.trim().toLowerCase();
  return !lower || texts.some((t) => (t ?? "").toLowerCase().includes(lower));
};
