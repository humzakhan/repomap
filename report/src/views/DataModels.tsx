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
import type { ModuleSummary, ExportedSymbol, RepoGraph } from "../types";
import { getLayerTheme, getLayerColor } from "../utils/layerColors";

interface DataModelsProps {
  summaries: ModuleSummary[];
  graph: RepoGraph;
  onNodeSelect: (nodeId: string) => void;
}

interface ModelInfo {
  name: string;
  kind: "class" | "interface" | "type" | "variable" | "constant" | "function";
  filePath: string;
  layer: string;
  fields: ExportedSymbol[];
  relatedModels: string[];
}

function extractModels(summaries: ModuleSummary[]): ModelInfo[] {
  const models: ModelInfo[] = [];
  const modelKinds = new Set(["class", "interface", "type"]);

  for (const s of summaries) {
    const typeExports = s.exports.filter((e) => modelKinds.has(e.kind));
    if (typeExports.length === 0) continue;

    for (const exp of typeExports) {
      // Find related models by checking dependencies
      const relatedModels: string[] = [];
      for (const dep of s.dependencies_on) {
        // Look for type references in dependencies
        const depSummary = summaries.find((ds) => ds.file_path === dep);
        if (depSummary) {
          for (const depExp of depSummary.exports) {
            if (modelKinds.has(depExp.kind)) {
              relatedModels.push(depExp.name);
            }
          }
        }
      }

      models.push({
        name: exp.name,
        kind: exp.kind as ModelInfo["kind"],
        filePath: s.file_path,
        layer: s.layer,
        fields: s.exports.filter((e) => e !== exp),
        relatedModels,
      });
    }
  }

  return models;
}

type ModelNodeData = {
  model: ModelInfo;
  selected: boolean;
};

function ModelNode({ data }: NodeProps<Node<ModelNodeData>>) {
  const { model, selected } = data;
  const colors = getLayerTheme(model.layer);

  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3 }}
      style={{
        minWidth: 180,
        borderRadius: 10,
        background: colors.bg,
        border: `1.5px solid ${selected ? colors.border : colors.border + "66"}`,
        boxShadow: selected ? `0 0 0 3px ${colors.border}44` : "0 2px 8px #00000040",
        cursor: "pointer",
        overflow: "hidden",
      }}
    >
      <Handle type="target" position={Position.Top} style={{ opacity: 0 }} />
      {/* Header */}
      <div
        style={{
          padding: "8px 12px",
          borderBottom: `1px solid ${colors.border}33`,
          display: "flex",
          alignItems: "center",
          gap: 6,
        }}
      >
        <span
          style={{
            fontSize: 9,
            fontFamily: "monospace",
            color: colors.border,
            background: colors.border + "18",
            padding: "1px 5px",
            borderRadius: 3,
            textTransform: "uppercase",
          }}
        >
          {model.kind}
        </span>
        <span
          style={{
            color: colors.text,
            fontSize: 12,
            fontFamily: "'JetBrains Mono', monospace",
            fontWeight: 700,
          }}
        >
          {model.name}
        </span>
      </div>
      {/* File */}
      <div
        style={{
          padding: "4px 12px",
          fontSize: 9,
          fontFamily: "monospace",
          color: "#475569",
        }}
      >
        {model.filePath.split("/").pop()}
      </div>
      {/* Fields preview */}
      {model.fields.length > 0 && (
        <div style={{ padding: "0 12px 8px" }}>
          {model.fields.slice(0, 4).map((f, i) => (
            <div
              key={i}
              style={{
                fontSize: 9,
                fontFamily: "monospace",
                color: "#94a3b8",
                padding: "1px 0",
              }}
            >
              {f.name}
              <span style={{ color: "#475569" }}> : {f.kind}</span>
            </div>
          ))}
          {model.fields.length > 4 && (
            <div style={{ fontSize: 9, color: "#475569", fontStyle: "italic" }}>
              +{model.fields.length - 4} more
            </div>
          )}
        </div>
      )}
      <Handle type="source" position={Position.Bottom} style={{ opacity: 0 }} />
    </motion.div>
  );
}

const nodeTypes: NodeTypes = { modelNode: ModelNode };

export function DataModels({ summaries, graph, onNodeSelect }: DataModelsProps) {
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);

  const models = useMemo(() => extractModels(summaries), [summaries]);

  const { initialNodes, initialEdges } = useMemo(() => {
    const cols = Math.max(1, Math.ceil(Math.sqrt(models.length)));
    const nodes: Node<ModelNodeData>[] = models.map((m, i) => ({
      id: `model-${m.name}`,
      type: "modelNode",
      position: { x: (i % cols) * 240, y: Math.floor(i / cols) * 180 },
      data: { model: m, selected: `model-${m.name}` === selectedNodeId },
    }));

    // Build edges between related models
    const nodeIds = new Set(nodes.map((n) => n.id));
    const edgeSet = new Set<string>();
    const edges: Edge[] = [];

    for (const model of models) {
      for (const related of model.relatedModels) {
        const targetId = `model-${related}`;
        const edgeKey = `model-${model.name}->${targetId}`;
        if (nodeIds.has(targetId) && !edgeSet.has(edgeKey)) {
          edgeSet.add(edgeKey);
          edges.push({
            id: edgeKey,
            source: `model-${model.name}`,
            target: targetId,
            style: { stroke: "#334155", strokeWidth: 1, opacity: 0.5 },
            markerEnd: {
              type: MarkerType.ArrowClosed,
              color: "#334155",
              width: 12,
              height: 8,
            },
          });
        }
      }
    }

    return { initialNodes: nodes, initialEdges: edges };
  }, [models, selectedNodeId]);

  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);

  React.useEffect(() => {
    setNodes(initialNodes);
    setEdges(initialEdges);
  }, [initialNodes, initialEdges, setNodes, setEdges]);

  const handleNodeClick = useCallback(
    (_: React.MouseEvent, node: Node) => {
      setSelectedNodeId(node.id);
      const model = models.find((m) => `model-${m.name}` === node.id);
      if (model) onNodeSelect(model.filePath);
    },
    [models, onNodeSelect],
  );

  if (models.length === 0) {
    return (
      <div className="canvas__empty">
        <div style={{ textAlign: "center" }}>
          <div style={{ fontSize: 16, fontWeight: 600, marginBottom: 8 }}>No Data Models Found</div>
          <div style={{ color: "#64748b", maxWidth: 400 }}>
            No class, interface, or type exports were detected in the analysis.
          </div>
        </div>
      </div>
    );
  }

  return (
    <div style={{ width: "100%", height: "100%", position: "relative" }}>
      <div
        style={{
          position: "absolute",
          top: 12,
          left: 12,
          zIndex: 10,
          background: "#111118ee",
          padding: "6px 12px",
          borderRadius: 8,
          border: "1px solid #1e1e2e",
          fontSize: 11,
          color: "#94a3b8",
          fontFamily: "monospace",
        }}
      >
        {models.length} model{models.length !== 1 ? "s" : ""} detected
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
        fitViewOptions={{ padding: 0.3 }}
        minZoom={0.2}
        maxZoom={3}
        proOptions={{ hideAttribution: true }}
        style={{ background: "#0a0a0f" }}
      >
        <Background color="#1e1e2e" gap={32} size={1} />
        <Controls showInteractive={false} style={{ background: "#111118", border: "1px solid #1e1e2e", borderRadius: 8 }} />
        <MiniMap
          nodeColor={() => getLayerColor("data")}
          maskColor="#0a0a0f88"
          style={{ background: "#111118", border: "1px solid #1e1e2e", borderRadius: 8 }}
        />
      </ReactFlow>
    </div>
  );
}
