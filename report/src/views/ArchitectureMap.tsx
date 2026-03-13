import React, { useMemo, useCallback, useState } from "react";
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
import type { RepoGraph, ModuleSummary, LayerType } from "../types";
import { groupIntoModules, type ModuleNode } from "../utils/moduleGrouper";
import { getLayerTheme, getLayerColor } from "../utils/layerColors";

interface ArchitectureMapProps {
  graph: RepoGraph;
  summaries: ModuleSummary[];
  onNodeSelect: (moduleId: string | null, moduleNode?: ModuleNode) => void;
}

// Layer group ordering for dagre-style layout
const LAYER_ORDER: Record<string, number> = {
  api: 0,
  service: 1,
  data: 2,
  config: 3,
  util: 4,
  test: 5,
  unknown: 6,
};

type ModuleNodeData = {
  moduleNode: ModuleNode;
  selected: boolean;
};

function ModuleCard({ data }: NodeProps<Node<ModuleNodeData>>) {
  const { moduleNode, selected } = data;
  const colors = getLayerTheme(moduleNode.layer);

  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.8 }}
      animate={{ opacity: 1, scale: 1 }}
      transition={{ duration: 0.3, ease: "easeOut" }}
      style={{
        minWidth: 160,
        padding: "10px 16px",
        borderRadius: 12,
        background: colors.bg,
        border: `1.5px solid ${selected ? colors.border : colors.border + "88"}`,
        boxShadow: selected
          ? `0 0 0 3px ${colors.border}44, 0 8px 32px #00000080`
          : "0 2px 8px #00000040",
        cursor: "pointer",
        transition: "border-color 0.15s, box-shadow 0.15s",
      }}
    >
      <Handle type="target" position={Position.Top} style={{ opacity: 0 }} />
      <div style={{ display: "flex", alignItems: "center", gap: 7, marginBottom: 4 }}>
        <div
          style={{
            width: 7,
            height: 7,
            borderRadius: "50%",
            background: colors.dot,
            flexShrink: 0,
          }}
        />
        <span
          style={{
            color: colors.text,
            fontSize: 11,
            fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
            fontWeight: 600,
            letterSpacing: "0.02em",
          }}
        >
          {moduleNode.label}
        </span>
      </div>
      <div style={{ display: "flex", gap: 6, alignItems: "center" }}>
        <span style={{ color: "#475569", fontSize: 10, fontFamily: "monospace" }}>
          {moduleNode.fileCount} file{moduleNode.fileCount !== 1 ? "s" : ""}
        </span>
        <span
          style={{
            color: colors.border,
            fontSize: 9,
            fontFamily: "monospace",
            background: colors.border + "18",
            padding: "1px 5px",
            borderRadius: 3,
          }}
        >
          {moduleNode.layer}
        </span>
      </div>
      <Handle type="source" position={Position.Bottom} style={{ opacity: 0 }} />
    </motion.div>
  );
}

const nodeTypes: NodeTypes = {
  moduleCard: ModuleCard,
};

export function ArchitectureMap({ graph, summaries, onNodeSelect }: ArchitectureMapProps) {
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [filterLayer, setFilterLayer] = useState<string | null>(null);

  const moduleGraph = useMemo(() => groupIntoModules(summaries, graph), [summaries, graph]);

  const nodeMap = useMemo(() => {
    const map = new Map<string, ModuleNode>();
    for (const n of moduleGraph.nodes) map.set(n.id, n);
    return map;
  }, [moduleGraph]);

  // Build React Flow nodes with simple hierarchical layout
  const { initialNodes, initialEdges } = useMemo(() => {
    // Group nodes by layer
    const layerGroups = new Map<string, ModuleNode[]>();
    for (const node of moduleGraph.nodes) {
      const layer = node.layer || "unknown";
      if (!layerGroups.has(layer)) layerGroups.set(layer, []);
      layerGroups.get(layer)!.push(node);
    }

    // Sort layers by order
    const sortedLayers = Array.from(layerGroups.entries()).sort(
      ([a], [b]) => (LAYER_ORDER[a] ?? 99) - (LAYER_ORDER[b] ?? 99),
    );

    const nodes: Node<ModuleNodeData>[] = [];
    let yOffset = 0;

    for (const [, layerNodes] of sortedLayers) {
      const rowWidth = layerNodes.length * 220;
      const startX = -rowWidth / 2 + 110;

      for (let i = 0; i < layerNodes.length; i++) {
        const mn = layerNodes[i];
        if (filterLayer && mn.layer !== filterLayer) continue;

        nodes.push({
          id: mn.id,
          type: "moduleCard",
          position: { x: startX + i * 220, y: yOffset },
          data: { moduleNode: mn, selected: mn.id === selectedNodeId },
        });
      }

      yOffset += 140;
    }

    // Build edges
    const visibleNodeIds = new Set(nodes.map((n) => n.id));
    const edges: Edge[] = moduleGraph.edges
      .filter((e) => visibleNodeIds.has(e.source) && visibleNodeIds.has(e.target))
      .map((edge) => ({
        id: `${edge.source}->${edge.target}`,
        source: edge.source,
        target: edge.target,
        animated: false,
        style: { stroke: "#1e293b", strokeWidth: 1.5, opacity: 0.5 },
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: "#334155",
          width: 16,
          height: 12,
        },
      }));

    return { initialNodes: nodes, initialEdges: edges };
  }, [moduleGraph, selectedNodeId, filterLayer]);

  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);

  // Sync when initial data changes
  React.useEffect(() => {
    setNodes(initialNodes);
    setEdges(initialEdges);
  }, [initialNodes, initialEdges, setNodes, setEdges]);

  const handleNodeClick = useCallback(
    (_: React.MouseEvent, node: Node) => {
      setSelectedNodeId(node.id);
      onNodeSelect(node.id, nodeMap.get(node.id));
    },
    [nodeMap, onNodeSelect],
  );

  const handlePaneClick = useCallback(() => {
    setSelectedNodeId(null);
    onNodeSelect(null);
  }, [onNodeSelect]);

  // Get unique layers for filter
  const availableLayers = useMemo(() => {
    const layers = new Set<string>();
    for (const n of moduleGraph.nodes) layers.add(n.layer);
    return Array.from(layers).sort(
      (a, b) => (LAYER_ORDER[a] ?? 99) - (LAYER_ORDER[b] ?? 99),
    );
  }, [moduleGraph]);

  return (
    <div style={{ width: "100%", height: "100%", position: "relative" }}>
      {/* Layer filter bar */}
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
        <FilterButton
          label="All"
          active={filterLayer === null}
          color="#94a3b8"
          onClick={() => setFilterLayer(null)}
        />
        {availableLayers.map((layer) => (
          <FilterButton
            key={layer}
            label={layer}
            active={filterLayer === layer}
            color={getLayerColor(layer)}
            onClick={() => setFilterLayer(filterLayer === layer ? null : layer)}
          />
        ))}
      </div>

      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onNodeClick={handleNodeClick}
        onPaneClick={handlePaneClick}
        nodeTypes={nodeTypes}
        fitView
        fitViewOptions={{ padding: 0.2 }}
        minZoom={0.2}
        maxZoom={3}
        proOptions={{ hideAttribution: true }}
        style={{ background: "#0a0a0f" }}
      >
        <Background color="#1e1e2e" gap={32} size={1} />
        <Controls
          showInteractive={false}
          style={{
            background: "#111118",
            border: "1px solid #1e1e2e",
            borderRadius: 8,
          }}
        />
        <MiniMap
          nodeColor={(n) => {
            const mn = nodeMap.get(n.id);
            return mn ? getLayerColor(mn.layer) : "#475569";
          }}
          maskColor="#0a0a0f88"
          style={{
            background: "#111118",
            border: "1px solid #1e1e2e",
            borderRadius: 8,
          }}
        />
      </ReactFlow>
    </div>
  );
}

function FilterButton({
  label,
  active,
  color,
  onClick,
}: {
  label: string;
  active: boolean;
  color: string;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      style={{
        display: "flex",
        alignItems: "center",
        gap: 4,
        padding: "3px 8px",
        borderRadius: 4,
        border: "none",
        background: active ? color + "33" : "transparent",
        color: active ? "#f1f5f9" : "#64748b",
        cursor: "pointer",
        fontSize: 10,
        fontFamily: "monospace",
        textTransform: "uppercase",
        transition: "all 0.15s",
      }}
    >
      <span
        style={{
          width: 6,
          height: 6,
          borderRadius: "50%",
          background: color,
        }}
      />
      {label}
    </button>
  );
}
