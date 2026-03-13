import React, { useMemo, useState, useCallback } from "react";
import {
  ReactFlow,
  Background,
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
import type { CriticalPath, FlowStep } from "../types";
import { getLayerTheme } from "../utils/layerColors";

interface CallFlowProps {
  criticalPaths: CriticalPath[];
  onNodeSelect: (nodeId: string) => void;
}

type StepNodeData = {
  step: FlowStep;
  stepIndex: number;
  totalSteps: number;
  selected: boolean;
};

function StepNode({ data }: NodeProps<Node<StepNodeData>>) {
  const { step, stepIndex, totalSteps, selected } = data;

  // Infer layer from module name
  const layerHint = inferLayerFromModule(step.module);
  const colors = getLayerTheme(layerHint);

  const progress = totalSteps > 1 ? stepIndex / (totalSteps - 1) : 0;

  return (
    <motion.div
      initial={{ opacity: 0, x: -20 }}
      animate={{ opacity: 1, x: 0 }}
      transition={{ duration: 0.3, delay: stepIndex * 0.08 }}
      style={{
        minWidth: 200,
        borderRadius: 10,
        background: colors.bg,
        border: `1.5px solid ${selected ? colors.border : colors.border + "66"}`,
        boxShadow: selected
          ? `0 0 0 3px ${colors.border}44, 0 4px 16px #00000060`
          : "0 2px 8px #00000040",
        cursor: "pointer",
        overflow: "hidden",
      }}
    >
      <Handle type="target" position={Position.Top} style={{ opacity: 0 }} />
      {/* Step number bar */}
      <div
        style={{
          height: 3,
          background: `linear-gradient(90deg, ${colors.border}00, ${colors.border})`,
          width: `${progress * 100}%`,
        }}
      />
      <div style={{ padding: "10px 14px" }}>
        <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 4 }}>
          <span
            style={{
              width: 20,
              height: 20,
              borderRadius: "50%",
              background: colors.border + "33",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              fontSize: 10,
              fontWeight: 700,
              color: colors.dot,
              fontFamily: "monospace",
              flexShrink: 0,
            }}
          >
            {stepIndex + 1}
          </span>
          <span
            style={{
              color: colors.text,
              fontSize: 11,
              fontFamily: "'JetBrains Mono', monospace",
              fontWeight: 600,
            }}
          >
            {step.module.split("/").pop() || step.module}
          </span>
        </div>
        <div
          style={{
            fontSize: 10,
            fontFamily: "'JetBrains Mono', monospace",
            color: colors.dot,
            marginBottom: 2,
          }}
        >
          {step.action}
        </div>
        <div style={{ fontSize: 10, color: "#64748b", lineHeight: 1.4 }}>
          {step.description}
        </div>
      </div>
      <Handle type="source" position={Position.Bottom} style={{ opacity: 0 }} />
    </motion.div>
  );
}

const nodeTypes: NodeTypes = { stepNode: StepNode };

export function CallFlow({ criticalPaths, onNodeSelect }: CallFlowProps) {
  const [selectedPath, setSelectedPath] = useState<number>(0);
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);

  const currentPath = criticalPaths[selectedPath] || null;

  const { initialNodes, initialEdges } = useMemo(() => {
    if (!currentPath) return { initialNodes: [], initialEdges: [] };

    const nodes: Node<StepNodeData>[] = currentPath.steps.map((step, i) => ({
      id: `step-${i}`,
      type: "stepNode",
      position: { x: 0, y: i * 160 },
      data: {
        step,
        stepIndex: i,
        totalSteps: currentPath.steps.length,
        selected: `step-${i}` === selectedNodeId,
      },
    }));

    const edges: Edge[] = [];
    for (let i = 0; i < currentPath.steps.length - 1; i++) {
      edges.push({
        id: `step-${i}->step-${i + 1}`,
        source: `step-${i}`,
        target: `step-${i + 1}`,
        animated: true,
        style: { stroke: "#7c3aed", strokeWidth: 2, opacity: 0.6 },
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: "#7c3aed",
          width: 16,
          height: 12,
        },
      });
    }

    return { initialNodes: nodes, initialEdges: edges };
  }, [currentPath, selectedNodeId]);

  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);

  React.useEffect(() => {
    setNodes(initialNodes);
    setEdges(initialEdges);
  }, [initialNodes, initialEdges, setNodes, setEdges]);

  const handleNodeClick = useCallback(
    (_: React.MouseEvent, node: Node) => {
      setSelectedNodeId(node.id);
      const stepIndex = parseInt(node.id.replace("step-", ""), 10);
      const step = currentPath?.steps[stepIndex];
      if (step) onNodeSelect(step.module);
    },
    [currentPath, onNodeSelect],
  );

  if (criticalPaths.length === 0) {
    return (
      <div className="canvas__empty">
        <div style={{ textAlign: "center" }}>
          <div style={{ fontSize: 16, fontWeight: 600, marginBottom: 8 }}>
            No Call Flows Available
          </div>
          <div style={{ color: "#64748b", maxWidth: 400 }}>
            Run analysis with architecture synthesis enabled to generate critical path data.
          </div>
        </div>
      </div>
    );
  }

  return (
    <div style={{ width: "100%", height: "100%", position: "relative" }}>
      {/* Path selector */}
      <div
        style={{
          position: "absolute",
          top: 12,
          left: 12,
          zIndex: 10,
          background: "#111118ee",
          padding: "8px 12px",
          borderRadius: 8,
          border: "1px solid #1e1e2e",
          maxWidth: 400,
        }}
      >
        <div style={{ fontSize: 10, color: "#64748b", marginBottom: 6, fontFamily: "monospace", textTransform: "uppercase" }}>
          Critical Paths
        </div>
        <div style={{ display: "flex", flexDirection: "column", gap: 2 }}>
          {criticalPaths.map((path, i) => (
            <button
              key={i}
              onClick={() => {
                setSelectedPath(i);
                setSelectedNodeId(null);
              }}
              style={{
                padding: "4px 8px",
                borderRadius: 4,
                border: "none",
                background: i === selectedPath ? "#7c3aed33" : "transparent",
                color: i === selectedPath ? "#f1f5f9" : "#64748b",
                cursor: "pointer",
                fontSize: 11,
                textAlign: "left",
                fontFamily: "'JetBrains Mono', monospace",
                transition: "all 0.15s",
              }}
            >
              {path.name}
              <span style={{ color: "#475569", fontSize: 9, marginLeft: 8 }}>
                {path.steps.length} steps
              </span>
            </button>
          ))}
        </div>
      </div>

      {/* Path description */}
      {currentPath && (
        <div
          style={{
            position: "absolute",
            top: 12,
            right: 12,
            zIndex: 10,
            background: "#111118ee",
            padding: "8px 12px",
            borderRadius: 8,
            border: "1px solid #1e1e2e",
            maxWidth: 300,
            fontSize: 11,
            color: "#94a3b8",
            lineHeight: 1.5,
          }}
        >
          {currentPath.description}
        </div>
      )}

      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onNodeClick={handleNodeClick}
        onPaneClick={() => setSelectedNodeId(null)}
        nodeTypes={nodeTypes}
        fitView
        fitViewOptions={{ padding: 0.4 }}
        minZoom={0.3}
        maxZoom={2}
        proOptions={{ hideAttribution: true }}
        style={{ background: "#0a0a0f" }}
      >
        <Background color="#1e1e2e" gap={32} size={1} />
        <Controls
          showInteractive={false}
          style={{ background: "#111118", border: "1px solid #1e1e2e", borderRadius: 8 }}
        />
      </ReactFlow>
    </div>
  );
}

function inferLayerFromModule(modulePath: string): string {
  const lower = modulePath.toLowerCase();
  if (/\b(api|route|handler|controller|endpoint)\b/.test(lower)) return "api";
  if (/\b(service|worker|core|domain)\b/.test(lower)) return "service";
  if (/\b(model|schema|entity|db|migration|repository)\b/.test(lower)) return "data";
  if (/\b(config|infra|deploy)\b/.test(lower)) return "config";
  if (/\b(util|helper|lib|pkg|shared)\b/.test(lower)) return "util";
  if (/\b(test|spec)\b/.test(lower)) return "test";
  return "unknown";
}
