import * as d3 from "d3";
import type { PipelineStats, RepoGraph, ModuleSummary, LayerType } from "../types";

const LAYER_COLORS: Record<string, string> = {
  api: "#7c3aed",
  service: "#06b6d4",
  data: "#f59e0b",
  util: "#64748b",
  test: "#10b981",
  config: "#ec4899",
  unknown: "#94a3b8",
};

export interface ModelCanvasOptions {
  stats: PipelineStats;
  graph: RepoGraph;
  summaries: ModuleSummary[];
  onNodeSelect: (nodeId: string) => void;
}

export class ModelCanvas {
  private container: HTMLElement;
  private stats: PipelineStats;
  private graph: RepoGraph;
  private summaries: ModuleSummary[];
  private onNodeSelect: (nodeId: string) => void;

  constructor(options: ModelCanvasOptions) {
    this.stats = options.stats;
    this.graph = options.graph;
    this.summaries = options.summaries;
    this.onNodeSelect = options.onNodeSelect;
    this.container = document.createElement("div");
    this.container.className = "canvas__d3";
    this.container.style.overflow = "auto";
  }

  mount(parent: HTMLElement): void {
    parent.appendChild(this.container);
    this.renderLanguageBreakdown();
    this.renderFilesByLayer();
  }

  private renderLanguageBreakdown(): void {
    const title = document.createElement("div");
    title.className = "model-canvas__title";
    title.textContent = "Language Breakdown";
    this.container.appendChild(title);

    const subtitle = document.createElement("div");
    subtitle.className = "model-canvas__subtitle";
    subtitle.textContent = `${this.stats.total_files} files across ${this.stats.languages.length} languages`;
    this.container.appendChild(subtitle);

    const chartContainer = document.createElement("div");
    chartContainer.className = "model-canvas__chart";
    this.container.appendChild(chartContainer);

    const width = 600;
    const height = 300;
    const margin = { top: 20, right: 30, bottom: 40, left: 120 };

    const svg = d3
      .select(chartContainer)
      .append("svg")
      .attr("width", width)
      .attr("height", height)
      .attr("viewBox", `0 0 ${width} ${height}`);

    const data = this.stats.languages.sort((a, b) => b.percentage - a.percentage);

    const y = d3
      .scaleBand()
      .domain(data.map((d) => d.language))
      .range([margin.top, height - margin.bottom])
      .padding(0.3);

    const x = d3
      .scaleLinear()
      .domain([0, 100])
      .range([margin.left, width - margin.right]);

    // Color scale
    const colorScale = d3
      .scaleOrdinal<string>()
      .domain(data.map((d) => d.language))
      .range(["#7c3aed", "#06b6d4", "#f59e0b", "#10b981", "#ec4899", "#64748b"]);

    // Bars
    svg
      .selectAll("rect.bar")
      .data(data)
      .join("rect")
      .attr("class", "bar")
      .attr("x", margin.left)
      .attr("y", (d) => y(d.language)!)
      .attr("width", (d) => x(d.percentage) - margin.left)
      .attr("height", y.bandwidth())
      .attr("rx", 4)
      .attr("fill", (d) => colorScale(d.language))
      .attr("opacity", 0.85);

    // Labels
    svg
      .selectAll("text.label")
      .data(data)
      .join("text")
      .attr("class", "label")
      .attr("x", margin.left - 8)
      .attr("y", (d) => y(d.language)! + y.bandwidth() / 2)
      .attr("dy", "0.35em")
      .attr("text-anchor", "end")
      .attr("fill", "#f1f5f9")
      .attr("font-size", "12px")
      .text((d) => d.language);

    // Percentage labels
    svg
      .selectAll("text.pct")
      .data(data)
      .join("text")
      .attr("class", "pct")
      .attr("x", (d) => x(d.percentage) + 6)
      .attr("y", (d) => y(d.language)! + y.bandwidth() / 2)
      .attr("dy", "0.35em")
      .attr("fill", "#64748b")
      .attr("font-size", "11px")
      .text((d) => `${d.percentage.toFixed(1)}% (${d.file_count})`);
  }

  private renderFilesByLayer(): void {
    const title = document.createElement("div");
    title.className = "model-canvas__title";
    title.style.marginTop = "32px";
    title.textContent = "Modules by Layer";
    this.container.appendChild(title);

    const subtitle = document.createElement("div");
    subtitle.className = "model-canvas__subtitle";
    subtitle.textContent = "Size represents token count";
    this.container.appendChild(subtitle);

    const chartContainer = document.createElement("div");
    chartContainer.className = "model-canvas__chart";
    this.container.appendChild(chartContainer);

    const width = 700;
    const height = 400;

    // Build treemap data
    interface TreeNode {
      name: string;
      children?: TreeNode[];
      value?: number;
      layer?: LayerType;
      filePath?: string;
    }

    const layerGroups = new Map<string, TreeNode[]>();
    for (const s of this.summaries) {
      const layer = s.layer || "unknown";
      if (!layerGroups.has(layer)) layerGroups.set(layer, []);
      layerGroups.get(layer)!.push({
        name: s.file_path.split("/").pop() || s.file_path,
        value: s.token_count,
        layer: s.layer,
        filePath: s.file_path,
      });
    }

    const rootData: TreeNode = {
      name: "root",
      children: Array.from(layerGroups.entries()).map(([layer, children]) => ({
        name: layer,
        children,
      })),
    };

    const root = d3
      .hierarchy(rootData)
      .sum((d) => (d as TreeNode).value || 0)
      .sort((a, b) => (b.value || 0) - (a.value || 0));

    d3.treemap<TreeNode>().size([width, height]).padding(3).round(true)(root);

    const svg = d3
      .select(chartContainer)
      .append("svg")
      .attr("width", width)
      .attr("height", height)
      .attr("viewBox", `0 0 ${width} ${height}`);

    const leaves = root.leaves();

    const cells = svg
      .selectAll("g")
      .data(leaves)
      .join("g")
      .attr("transform", (d) => {
        const td = d as d3.HierarchyRectangularNode<TreeNode>;
        return `translate(${td.x0},${td.y0})`;
      })
      .style("cursor", "pointer")
      .on("click", (_event, d) => {
        const filePath = (d.data as TreeNode).filePath;
        if (filePath) {
          this.onNodeSelect(filePath);
        }
      });

    cells
      .append("rect")
      .attr("width", (d) => {
        const td = d as d3.HierarchyRectangularNode<TreeNode>;
        return Math.max(0, td.x1 - td.x0);
      })
      .attr("height", (d) => {
        const td = d as d3.HierarchyRectangularNode<TreeNode>;
        return Math.max(0, td.y1 - td.y0);
      })
      .attr("rx", 3)
      .attr("fill", (d) => {
        const layer = (d.data as TreeNode).layer || "unknown";
        return LAYER_COLORS[layer] || LAYER_COLORS.unknown;
      })
      .attr("opacity", 0.7)
      .on("mouseover", function () {
        d3.select(this).attr("opacity", 1);
      })
      .on("mouseout", function () {
        d3.select(this).attr("opacity", 0.7);
      });

    cells
      .append("text")
      .attr("x", 4)
      .attr("y", 14)
      .attr("fill", "#f1f5f9")
      .attr("font-size", "10px")
      .attr("font-weight", "500")
      .text((d) => {
        const td = d as d3.HierarchyRectangularNode<TreeNode>;
        const w = td.x1 - td.x0;
        const name = (d.data as TreeNode).name;
        if (w < 40) return "";
        if (w < 80) return name.substring(0, 5) + "...";
        return name;
      });
  }

  unmount(): void {
    this.container.remove();
  }

  getContainer(): HTMLElement {
    return this.container;
  }
}
