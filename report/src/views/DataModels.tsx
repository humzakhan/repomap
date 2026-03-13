import React from "react";
import type { ModuleSummary } from "../types";

interface DataModelsProps {
  summaries: ModuleSummary[];
  onNodeSelect: (nodeId: string) => void;
}

export function DataModels({ summaries, onNodeSelect }: DataModelsProps) {
  const dataModels = summaries.filter(
    (s) => s.layer === "data" || s.patterns.some((p) => p.toLowerCase().includes("model")),
  );

  return (
    <div className="canvas__empty">
      <div style={{ textAlign: "center" }}>
        <div style={{ fontSize: 48, marginBottom: 16 }}>{"\uD83D\uDDC3\uFE0F"}</div>
        <div style={{ fontSize: 16, fontWeight: 600, marginBottom: 8 }}>Data Models</div>
        <div style={{ color: "#64748b", maxWidth: 400 }}>
          ERD-style visualization of schemas, interfaces, types, and their relationships.
          Cross-linked across the codebase.
        </div>
        <div style={{ color: "#475569", marginTop: 12, fontSize: 12 }}>
          {dataModels.length} data-layer files detected
        </div>
      </div>
    </div>
  );
}
