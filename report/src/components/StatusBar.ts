import type { PipelineStats, RepoGraph } from "../types";

export interface StatusBarOptions {
  stats: PipelineStats;
  graph: RepoGraph;
}

export class StatusBar {
  private el: HTMLElement;

  constructor(options: StatusBarOptions) {
    const { stats, graph } = options;
    this.el = document.createElement("div");
    this.el.className = "status-bar";

    this.addItem(`${graph.nodes.length} nodes`);
    this.addItem(`${graph.edges.length} edges`);
    this.addItem(`${stats.total_modules} modules`);
    this.addItem(`${stats.total_files} files`);

    const langs = (stats.languages || [])
      .map((l) => `${l.language} ${l.percentage.toFixed(0)}%`)
      .join(", ");
    this.addItem(langs);

    // Spacer
    const spacer = document.createElement("div");
    spacer.className = "status-bar__spacer";
    this.el.appendChild(spacer);

    this.addItem(`Model: ${stats.model_used}`);
    this.addItem(`Cost: $${stats.total_cost.toFixed(2)}`);

    const duration = (stats.duration_ms / 1000).toFixed(1);
    this.addItem(`${duration}s`);
  }

  private addItem(text: string): void {
    const item = document.createElement("span");
    item.className = "status-bar__item";
    item.textContent = text;
    this.el.appendChild(item);
  }

  mount(parent: HTMLElement): void {
    parent.appendChild(this.el);
  }
}
