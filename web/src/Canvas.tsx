import {
  Background,
  Controls,
  MarkerType,
  MiniMap,
  Panel,
  ReactFlow,
  useReactFlow,
  useStore,
  type Edge,
  type Node,
  type NodeChange,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { useCallback, useEffect, useMemo, useState } from "react";
import { isUnknown, repoID, type Bundle, type Visible } from "./graph";
import { T } from "./i18n";
import { bestDirection, layout, sizeOf, type Point } from "./layout";
import { isFaded, type EdgeMark, type Marks, type NodeMark } from "./marks";
import { nodeTypes, type RepoData, type UnknownData } from "./nodes";
import type { Model } from "./types";

// With more arrows than this and the canvas zoomed out, only the marked or
// hovered arrows show their label.
const LABEL_LIMIT = 12;
const ZOOMED_IN = 1.2;
const LABEL_CHARS = 38;

const MARK_COLOR: Record<EdgeMark, string> = {
  direct: "var(--direct)",
  indirect: "var(--indirect)",
  possible: "var(--possible)",
  edge: "var(--sel)",
};
const MARK_WIDTH: Record<EdgeMark, number> = { direct: 3, indirect: 2.5, possible: 2.5, edge: 3.5 };

const NODE_COLOR: Record<NodeMark, string> = {
  selected: "var(--sel)",
  direct: "var(--direct)",
  indirect: "var(--indirect)",
  possible: "var(--possible)",
};

function lineStyle(group: string, mark: EdgeMark | undefined) {
  if (mark) return { stroke: MARK_COLOR[mark], strokeWidth: MARK_WIDTH[mark], strokeDasharray: undefined };
  switch (group) {
    case "declared":
      return { stroke: "var(--declared)", strokeWidth: 1.5, strokeDasharray: undefined };
    case "unverified":
      return { stroke: "var(--bad)", strokeWidth: 1.5, strokeDasharray: "2 4" };
    case "pending":
      return { stroke: "var(--edge)", strokeWidth: 1.5, strokeDasharray: "6 4" };
    default:
      return { stroke: "var(--edge)", strokeWidth: 1.5, strokeDasharray: undefined };
  }
}

const NOT_MOVED: Record<string, Point> = {};

const clip = (s: string) => (s.length > LABEL_CHARS ? `${s.slice(0, LABEL_CHARS - 1)}…` : s);

export type Reveal = { name: string; n: number };

type Props = {
  model: Model;
  visible: Visible;
  marks: Marks;
  reveal: Reveal | null;
  onSelectRepo: (name: string) => void;
  onSelectBundle: (id: string) => void;
  onClear: () => void;
};

function Legend() {
  return (
    <div className="legend">
      {(["selected", "direct", "indirect", "possible"] as const).map((k) => (
        <span key={k}>
          <span className={`swatch ${k}`} />
          {T.legend[k]}
        </span>
      ))}
    </div>
  );
}

export function Canvas({ model, visible, marks, reveal, onSelectRepo, onSelectBundle, onClear }: Props) {
  const flow = useReactFlow();
  const [zoomedIn, setZoomedIn] = useState(false);
  const [hover, setHover] = useState<string | null>(null);

  // The layout changes only when what is on the canvas changes, not when a
  // selection marks it: clicking never moves anything. The direction
  // follows the shape of the canvas, so the graph is drawn as large as it
  // can be.
  const width = useStore((s) => s.width);
  const height = useStore((s) => s.height);
  const shape = `${visible.nodes.join(",")}|${visible.edges.map((b) => b.id).join(",")}`;
  const lr = useMemo(() => layout(visible.nodes, visible.edges, "LR"), [shape]); // eslint-disable-line react-hooks/exhaustive-deps
  const tb = useMemo(() => layout(visible.nodes, visible.edges, "TB"), [shape]); // eslint-disable-line react-hooks/exhaustive-deps
  const dir = bestDirection(lr, tb, { w: width, h: height });
  const positions = (dir === "TB" ? tb : lr).at;
  const key = `${dir}|${shape}`;
  const [dragged, setDragged] = useState<{ key: string; at: Record<string, Point> }>({ key, at: {} });
  const moved = dragged.key === key ? dragged.at : NOT_MOVED;

  const repos = useMemo(() => new Map(model.repos.map((r) => [r.name, r])), [model]);

  const nodes = useMemo<Node[]>(
    () =>
      visible.nodes.map((id) => {
        const { w, h } = sizeOf(id);
        const base = { id, position: moved[id] ?? positions.get(id) ?? { x: 0, y: 0 }, width: w, height: h, selectable: false };
        const faded = isFaded(marks, id);
        const mark = marks.nodes.get(id);
        if (isUnknown(id)) return { ...base, type: "unknown", data: { mark, faded, dir } satisfies UnknownData };
        const repo = repos.get(id.slice(2));
        return { ...base, type: "repo", data: { repo, mark, faded, dir } as RepoData };
      }),
    [visible.nodes, positions, moved, marks, repos, dir],
  );

  const many = visible.edges.length > LABEL_LIMIT && !zoomedIn;
  const edges = useMemo<Edge[]>(
    () =>
      visible.edges.map((b: Bundle) => {
        const mark = marks.edges.get(b.id);
        const line = lineStyle(b.group, mark);
        const text = b.idxs.length === 1 ? model.edges[b.idxs[0]].id : T.bundle(b.idxs.length, T.kind[b.group]);
        const showLabel = !many || !!mark || hover === b.id;
        return {
          id: b.id,
          source: b.source,
          target: b.target,
          label: showLabel ? clip(text) : undefined,
          markerEnd: { type: MarkerType.ArrowClosed, color: line.stroke, width: 16, height: 16 },
          style: line,
          className: isFaded(marks, b.id) ? "faded" : undefined,
          zIndex: mark ? 10 : 0,
          interactionWidth: 20,
          selectable: false,
        };
      }),
    [visible.edges, marks, model, many, hover],
  );

  const onNodesChange = useCallback(
    (changes: NodeChange[]) => {
      const at = { ...moved };
      let any = false;
      for (const c of changes) {
        if (c.type === "position" && c.position) {
          at[c.id] = c.position;
          any = true;
        }
      }
      if (any) setDragged({ key, at });
    },
    [moved, key],
  );

  // Fit whatever the layout now holds, on the first paint and whenever the
  // filters change it. The canvas has no size until it is laid out, so wait
  // a frame for it.
  useEffect(() => {
    const id = requestAnimationFrame(() => flow.fitView({ padding: 0.12, maxZoom: ZOOMED_IN, duration: 0 }));
    return () => cancelAnimationFrame(id);
  }, [key]); // eslint-disable-line react-hooks/exhaustive-deps

  // Bring a repository found by search, or picked in the drawer, to the
  // middle of the canvas without zooming out from where the reader is.
  useEffect(() => {
    if (!reveal) return;
    const node = flow.getNode(repoID(reveal.name));
    if (!node) return;
    const { w, h } = sizeOf(node.id);
    flow.setCenter(node.position.x + w / 2, node.position.y + h / 2, { zoom: Math.max(flow.getZoom(), 0.8), duration: 300 });
  }, [reveal]); // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <ReactFlow
      nodes={nodes}
      edges={edges}
      nodeTypes={nodeTypes}
      onNodesChange={onNodesChange}
      onNodeClick={(_, node) => {
        if (node.type === "repo") onSelectRepo(node.id.slice(2));
      }}
      onEdgeClick={(_, edge) => onSelectBundle(edge.id)}
      onPaneClick={onClear}
      onEdgeMouseEnter={(_, edge) => setHover(edge.id)}
      onEdgeMouseLeave={() => setHover(null)}
      onMove={(_, vp) => setZoomedIn(vp.zoom >= ZOOMED_IN)}
      minZoom={0.1}
      maxZoom={3}
      nodesConnectable={false}
      elementsSelectable={false}
      edgesFocusable={false}
      colorMode="system"
      proOptions={{ hideAttribution: false }}
      deleteKeyCode={null}
    >
      <Background gap={24} size={1} />
      <Controls showInteractive={false} fitViewOptions={{ padding: 0.12, maxZoom: ZOOMED_IN, duration: 250 }} />
      <MiniMap
        pannable
        zoomable
        style={{ width: 150, height: 100 }}
        nodeColor={(n) => {
          const mark = (n.data as { mark?: NodeMark }).mark;
          return mark ? NODE_COLOR[mark] : "var(--node-border)";
        }}
        maskColor="var(--minimap-mask)"
      />
      <Panel position="bottom-center">
        <Legend />
      </Panel>
    </ReactFlow>
  );
}
