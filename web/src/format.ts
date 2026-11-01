import { T } from "./i18n";
import type { Repo } from "./types";

export function ago(iso: string): string {
  const days = Math.floor((Date.now() - Date.parse(iso)) / 86400000);
  if (Number.isNaN(days)) return "";
  return days <= 0 ? T.today : T.daysAgo(days);
}

export function releaseText(r: Repo): string {
  if (r.release) return `${r.release.version} · ${ago(r.release.date)}`;
  return r.cloned ? T.noRelease : T.notCloned;
}

// Only https links from the model become anchors; the rest is plain text.
export const isLink = (url: string | undefined): url is string => typeof url === "string" && url.startsWith("https://");
