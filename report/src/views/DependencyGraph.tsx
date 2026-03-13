import React from "react";
import type { RepoGraph, ModuleSummary } from "../types";

interface DependencyGraphProps {
  graph: RepoGraph;
  summaries: ModuleSummary[];
  onNodeSelect: (nodeId: string) => void;
}

export function DependencyGraph({ graph, summaries, onNodeSelect }: DependencyGraphProps) {
  return (
    <div className="canvas__empty">
      <div style={{ textAlign: "center" }}>
        <div style={{ fontSize: 48, marginBottom: 16 }}>{"\uD83D\uDD78\uFE0F"}</div>
        <div style={{ fontSize: 16, fontWeight: 600, marginBottom: 8 }}>Dependency Graph</div>
        <div style={{ color: "#64748b", maxWidth: 400 }}>
          Interactive import-relationship explorer with circular dependency detection,
          fan-in/fan-out metrics, and dead code identification.
        </div>
        <div style={{ color: "#475569", marginTop: 12, fontSize: 12 }}>
          {graph.nodes.length} nodes, {graph.edges.length} edges ready
        </div>
      </div>
    </div>
  );
}
