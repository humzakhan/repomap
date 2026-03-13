import React, { useMemo, useState, useCallback } from "react";
import {
  ReactFlow,
  Background,
  MiniMap,
  Controls,
  type Node,
  type Edge,
  type NodeTypes,
  type NodeProps,
  useNodesState,
  useEdgesState,
  MarkerType,
  Handle,
  Position,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { motion } from "framer-motion";
import type { RepoGraph, ModuleSummary } from "../types";
import { getLayerColor, getLayerTheme } from "../utils/layerColors";

interface DependencyGraphProps {
  graph: RepoGraph;
  summaries: ModuleSummary[];
  onNodeSelect: (nodeId: string) => void;
}

interface FileMetrics {
  id: string;
  label: string;
  layer: string;
  fanIn: number;
  fanOut: number;
  isCircular: boolean;
  isDead: boolean;
}

function analyzeGraph(graph: RepoGraph): {
  metrics: Map<string, FileMetrics>;
  circularEdges: Set<string>;
} {
  const fanIn = new Map<string, number>();
  const fanOut = new Map<string, number>();
  const adjacency = new Map<string, Set<string>>();

  for (const node of graph.nodes) {
    fanIn.set(node.id, 0);
    fanOut.set(node.id, 0);
    adjacency.set(node.id, new Set());
  }

  for (const edge of graph.edges) {
    fanOut.set(edge.source, (fanOut.get(edge.source) || 0) + 1);
    fanIn.set(edge.target, (fanIn.get(edge.target) || 0) + 1);
    adjacency.get(edge.source)?.add(edge.target);
  }

  const circularNodes = new Set<string>();
  const circularEdges = new Set<string>();
  const visited = new Set<string>();
  const stack = new Set<string>();

  function dfs(nodeId: string) {
    visited.add(nodeId);
    stack.add(nodeId);
    for (const neighbor of adjacency.get(nodeId) || []) {
      if (stack.has(neighbor)) {
        circularNodes.add(nodeId);
        circularNodes.add(neighbor);
        circularEdges.add(`${nodeId}->${neighbor}`);
      } else if (!visited.has(neighbor)) {
        dfs(neighbor);
      }
    }
    stack.delete(nodeId);
  }

  for (const node of graph.nodes) {
    if (!visited.has(node.id)) dfs(node.id);
  }

  const entryPatterns = [
    /main\.(ts|js|go|py|rs)$/,
    /index\.(ts|js|tsx|jsx)$/,
    /app\.(ts|js|tsx|jsx)$/,
    /__init__\.py$/,
    /cmd\//,
  ];

  const metrics = new Map<string, FileMetrics>();
  for (const node of graph.nodes) {
    const inCount = fanIn.get(node.id) || 0;
    const outCount = fanOut.get(node.id) || 0;
    const isEntry = entryPatterns.some((p) => p.test(node.id));
    metrics.set(node.id, {
      id: node.id,
      label: (node.path || node.id).split("/").pop() || node.id,
      layer: node.layer || "unknown",
      fanIn: inCount,
      fanOut: outCount,
      isCircular: circularNodes.has(node.id),
      isDead: inCount === 0 && outCount > 0 && !isEntry,
    });
  }

  return { metrics, circularEdges };
}

type FileNodeData = {
  metrics: FileMetrics;
  selected: boolean;
  highlightMode: "normal" | "circular" | "dead";
};

function FileNode({ data }: NodeProps<Node<FileNodeData>>) {
  const { metrics, selected, highlightMode } = data;
  const colors = getLayerTheme(metrics.layer);

  const borderColor =
    highlightMode === "circular"
      ? "#ef4444"
      : highlightMode === "dead"
        ? "#f59e0b88"
        : selected
          ? colors.border
          : colors.border + "66";

  const glowColor =
    highlightMode === "circular"
      ? "0 0 0 3px #ef444444"
      : highlightMode === "dead"
        ? "0 0 0 3px #f59e0b22"
        : selected
          ? `0 0 0 3px ${colors.border}44`
          : "none";

  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.8 }}
      animate={{ opacity: 1, scale: 1 }}
      transition={{ duration: 0.2 }}
      style={{
        padding: "8px 12px",
        borderRadius: 8,
        background: colors.bg,
        border: `1.5px solid ${borderColor}`,
        boxShadow: glowColor,
        cursor: "pointer",
        minWidth: 120,
      }}
    >
      <Handle type="target" position={Position.Top} style={{ opacity: 0 }} />
      <div style={{ display: "flex", alignItems: "center", gap: 6, marginBottom: 3 }}>
        <div
          style={{ width: 6, height: 6, borderRadius: "50%", background: colors.dot }}
        />
        <span
          style={{
            color: colors.text,
            fontSize: 10,
            fontFamily: "'JetBrains Mono', monospace",
            fontWeight: 600,
          }}
        >
          {metrics.label}
        </span>
      </div>
      <div style={{ display: "flex", gap: 8, fontSize: 9, fontFamily: "monospace" }}>
        <span style={{ color: "#22d3ee" }}>{"\u2190"}{metrics.fanIn}</span>
        <span style={{ color: "#a78bfa" }}>{"\u2192"}{metrics.fanOut}</span>
        {metrics.isCircular && (
          <span style={{ color: "#ef4444", fontWeight: 700 }}>{"\u26A0"} cycle</span>
        )}
        {metrics.isDead && <span style={{ color: "#f59e0b" }}>{"\u26A0"} unused</span>}
      </div>
      <Handle type="source" position={Position.Bottom} style={{ opacity: 0 }} />
    </motion.div>
  );
}

const nodeTypes: NodeTypes = { fileNode: FileNode };

type ViewMode = "all" | "circular" | "dead";

export function DependencyGraph({ graph, summaries, onNodeSelect }: DependencyGraphProps) {
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [viewMode, setViewMode] = useState<ViewMode>("all");

  const { metrics, circularEdges } = useMemo(() => analyzeGraph(graph), [graph]);

  const circularCount = useMemo(() => {
    let count = 0;
    for (const [, m] of metrics) if (m.isCircular) count++;
    return count;
  }, [metrics]);

  const deadCount = useMemo(() => {
    let count = 0;
    for (const [, m] of metrics) if (m.isDead) count++;
    return count;
  }, [metrics]);

  const { initialNodes, initialEdges } = useMemo(() => {
    const visibleNodes = Array.from(metrics.values()).filter((m) => {
      if (viewMode === "circular") return m.isCircular;
      if (viewMode === "dead") return m.isDead;
      return true;
    });

    const cols = Math.max(1, Math.ceil(Math.sqrt(visibleNodes.length)));
    const nodes: Node<FileNodeData>[] = visibleNodes.map((m, i) => ({
      id: m.id,
      type: "fileNode",
      position: { x: (i % cols) * 200, y: Math.floor(i / cols) * 100 },
      data: {
        metrics: m,
        selected: m.id === selectedNodeId,
        highlightMode:
          m.isCircular && viewMode !== "dead"
            ? "circular" as const
            : m.isDead
              ? "dead" as const
              : "normal" as const,
      },
    }));

    const visibleIds = new Set(visibleNodes.map((m) => m.id));
    const edges: Edge[] = graph.edges
      .filter((e) => visibleIds.has(e.source) && visibleIds.has(e.target))
      .map((edge) => {
        const isCircular = circularEdges.has(`${edge.source}->${edge.target}`);
        return {
          id: `${edge.source}->${edge.target}`,
          source: edge.source,
          target: edge.target,
          animated: isCircular,
          style: {
            stroke: isCircular ? "#ef4444" : "#1e293b",
            strokeWidth: isCircular ? 2 : 1,
            opacity: isCircular ? 0.8 : 0.4,
          },
          markerEnd: {
            type: MarkerType.ArrowClosed,
            color: isCircular ? "#ef4444" : "#334155",
            width: 14,
            height: 10,
          },
        };
      });

    return { initialNodes: nodes, initialEdges: edges };
  }, [metrics, graph, selectedNodeId, viewMode, circularEdges]);

  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);

  React.useEffect(() => {
    setNodes(initialNodes);
    setEdges(initialEdges);
  }, [initialNodes, initialEdges, setNodes, setEdges]);

  const handleNodeClick = useCallback(
    (_: React.MouseEvent, node: Node) => {
      setSelectedNodeId(node.id);
      onNodeSelect(node.id);
    },
    [onNodeSelect],
  );

  return (
    <div style={{ width: "100%", height: "100%", position: "relative" }}>
      <div
        style={{
          position: "absolute",
          top: 12,
          left: 12,
          zIndex: 10,
          display: "flex",
          gap: 4,
          background: "#111118ee",
          padding: "4px 8px",
          borderRadius: 8,
          border: "1px solid #1e1e2e",
        }}
      >
        <ModeButton label={`All (${graph.nodes.length})`} active={viewMode === "all"} onClick={() => setViewMode("all")} />
        <ModeButton label={`Circular (${circularCount})`} active={viewMode === "circular"} color="#ef4444" onClick={() => setViewMode("circular")} />
        <ModeButton label={`Unused (${deadCount})`} active={viewMode === "dead"} color="#f59e0b" onClick={() => setViewMode("dead")} />
      </div>

      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onNodeClick={handleNodeClick}
        onPaneClick={() => setSelectedNodeId(null)}
        nodeTypes={nodeTypes}
        fitView
        fitViewOptions={{ padding: 0.2 }}
        minZoom={0.1}
        maxZoom={3}
        proOptions={{ hideAttribution: true }}
        style={{ background: "#0a0a0f" }}
      >
        <Background color="#1e1e2e" gap={32} size={1} />
        <Controls showInteractive={false} style={{ background: "#111118", border: "1px solid #1e1e2e", borderRadius: 8 }} />
        <MiniMap
          nodeColor={(n) => {
            const m = metrics.get(n.id);
            if (m?.isCircular) return "#ef4444";
            if (m?.isDead) return "#f59e0b";
            return getLayerColor(m?.layer || "unknown");
          }}
          maskColor="#0a0a0f88"
          style={{ background: "#111118", border: "1px solid #1e1e2e", borderRadius: 8 }}
        />
      </ReactFlow>
    </div>
  );
}

function ModeButton({ label, active, color = "#94a3b8", onClick }: { label: string; active: boolean; color?: string; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      style={{
        padding: "3px 8px",
        borderRadius: 4,
        border: "none",
        background: active ? color + "33" : "transparent",
        color: active ? "#f1f5f9" : "#64748b",
        cursor: "pointer",
        fontSize: 10,
        fontFamily: "monospace",
        transition: "all 0.15s",
      }}
    >
      {label}
    </button>
  );
}
