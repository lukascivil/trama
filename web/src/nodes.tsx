import { Handle, Position, type Node, type NodeProps } from "@xyflow/react";
import { releaseText } from "./format";
import { T } from "./i18n";
import type { Direction } from "./layout";
import type { NodeMark } from "./marks";
import type { Repo } from "./types";
import { Chip } from "./ui";

// dir is the direction of the layout: arrows leave a repository from the
// side that faces the ones it consumes from.
export type RepoData = { repo: Repo; mark?: NodeMark; faded: boolean; dir: Direction };
export type UnknownData = { mark?: NodeMark; faded: boolean; dir: Direction };

const IN: Record<Direction, Position> = { LR: Position.Left, TB: Position.Top };
const OUT: Record<Direction, Position> = { LR: Position.Right, TB: Position.Bottom };

const cls = (...parts: (string | false | undefined)[]) => parts.filter(Boolean).join(" ");

// A repository: its name, its latest release and what kind of repository
// it is. The border says how it relates to the selection.
export function RepoNode({ data }: NodeProps<Node<RepoData, "repo">>) {
  const { repo, mark, faded, dir } = data;
  const changes = (repo.openChanges ?? []).length;
  const stale = repo.card === "stale" || repo.card === "missing";
  return (
    <div
      className={cls("rn", mark, faded && "faded", stale && "stale", changes > 0 && "change", !repo.cloned && "notcloned")}
      title={[repo.name, T.card[repo.card] && stale ? T.card[repo.card] : ""].filter(Boolean).join("\n")}
    >
      <Handle type="target" position={IN[dir]} isConnectable={false} />
      <div className="rn-name">{repo.name}</div>
      <div className="rn-meta">
        <Chip role={repo.role} />
        <span className="rn-release">{releaseText(repo)}</span>
        {changes > 0 && <span className="rn-dot" title={T.openChanges} />}
      </div>
      <Handle type="source" position={OUT[dir]} isConnectable={false} />
    </div>
  );
}

// The "?" a consumer's unproven relations point to.
export function UnknownNode({ data }: NodeProps<Node<UnknownData, "unknown">>) {
  return (
    <div className={cls("un", data.mark, data.faded && "faded")}>
      <Handle type="target" position={IN[data.dir]} isConnectable={false} />?
    </div>
  );
}

export const nodeTypes = { repo: RepoNode, unknown: UnknownNode };
