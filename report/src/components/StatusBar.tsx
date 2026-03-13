import React from "react";
import type { PipelineStats, RepoGraph } from "../types";

interface StatusBarProps {
  stats: PipelineStats;
  graph: RepoGraph;
}

export function StatusBar({ stats, graph }: StatusBarProps) {
  const langs = (stats.languages || [])
    .map((l) => `${l.language} ${l.percentage.toFixed(0)}%`)
    .join(", ");
  const duration = (stats.duration_ms / 1000).toFixed(1);

  return (
    <div className="status-bar">
      <Item text={`${graph.nodes.length} nodes`} />
      <Item text={`${graph.edges.length} edges`} />
      <Item text={`${stats.total_modules} modules`} />
      <Item text={`${stats.total_files} files`} />
      <Item text={langs} />
      <div className="status-bar__spacer" />
      <Item text={`Model: ${stats.model_used}`} />
      <Item text={`Cost: $${stats.total_cost.toFixed(2)}`} />
      <Item text={`${duration}s`} />
    </div>
  );
}

function Item({ text }: { text: string }) {
  return <span className="status-bar__item">{text}</span>;
}
