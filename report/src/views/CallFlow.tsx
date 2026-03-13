import React from "react";
import type { CriticalPath } from "../types";

interface CallFlowProps {
  criticalPaths: CriticalPath[];
  onNodeSelect: (nodeId: string) => void;
}

export function CallFlow({ criticalPaths, onNodeSelect }: CallFlowProps) {
  return (
    <div className="canvas__empty">
      <div style={{ textAlign: "center" }}>
        <div style={{ fontSize: 48, marginBottom: 16 }}>{"\uD83D\uDCE1"}</div>
        <div style={{ fontSize: 16, fontWeight: 600, marginBottom: 8 }}>Call Flow Tracer</div>
        <div style={{ color: "#64748b", maxWidth: 400 }}>
          Trace execution paths from entry points through the stack.
          Animated flow visualization with depth control.
        </div>
        <div style={{ color: "#475569", marginTop: 12, fontSize: 12 }}>
          {criticalPaths.length} critical paths available
        </div>
      </div>
    </div>
  );
}
