import cytoscape from "cytoscape";
import cytoscapeDagre from "cytoscape-dagre";
import type { RepoGraph, ModuleSummary, LayerType } from "../types";
import { groupIntoModules, type ModuleNode, type ModuleGraph } from "../utils/moduleGrouper";

// Register dagre layout
cytoscapeDagre(cytoscape);

const LAYER_COLORS: Record<string, { bg: string; border: string; dot: string; text: string; badge: string }> = {
  api:     { bg: "#1e1040", border: "#7c3aed", dot: "#a78bfa", text: "#c4b5fd", badge: "#7c3aed" },
  service: { bg: "#0e2a33", border: "#0891b2", dot: "#22d3ee", text: "#67e8f9", badge: "#0891b2" },
  data:    { bg: "#2a1a00", border: "#d97706", dot: "#fbbf24", text: "#fcd34d", badge: "#d97706" },
  config:  { bg: "#1a0a2e", border: "#db2777", dot: "#f472b6", text: "#fbcfe8", badge: "#db2777" },
  util:    { bg: "#1a1a24", border: "#64748b", dot: "#94a3b8", text: "#cbd5e1", badge: "#64748b" },
  test:    { bg: "#0a2a1a", border: "#059669", dot: "#34d399", text: "#6ee7b7", badge: "#059669" },
  unknown: { bg: "#1a1a24", border: "#475569", dot: "#94a3b8", text: "#cbd5e1", badge: "#475569" },
};

function getColors(layer: string) {
  return LAYER_COLORS[layer] || LAYER_COLORS.unknown;
}

export interface ArchitectureMapOptions {
  graph: RepoGraph;
  summaries: ModuleSummary[];
  onNodeSelect: (moduleId: string | null, moduleNode?: ModuleNode) => void;
}

export class ArchitectureMap {
  private cy: cytoscape.Core | null = null;
  private container: HTMLElement;
  private overlay: HTMLElement;
  private onNodeSelect: (moduleId: string | null, moduleNode?: ModuleNode) => void;
  private graph: RepoGraph;
  private summaries: ModuleSummary[];
  private moduleGraph: ModuleGraph;
  private nodeMap: Map<string, ModuleNode> = new Map();

  constructor(options: ArchitectureMapOptions) {
    this.graph = options.graph;
    this.summaries = options.summaries;
    this.onNodeSelect = options.onNodeSelect;

    this.container = document.createElement("div");
    this.container.className = "canvas__cytoscape";

    this.overlay = document.createElement("div");
    this.overlay.className = "canvas__node-overlay";
    this.overlay.style.cssText = "position:absolute;inset:0;pointer-events:none;overflow:hidden;";

    this.moduleGraph = groupIntoModules(this.summaries, this.graph);
    for (const n of this.moduleGraph.nodes) {
      this.nodeMap.set(n.id, n);
    }
  }

  mount(parent: HTMLElement): void {
    parent.appendChild(this.container);
    parent.appendChild(this.overlay);
    this.initCytoscape();
  }

  private initCytoscape(): void {
    const elements: cytoscape.ElementDefinition[] = [];

    for (const node of this.moduleGraph.nodes) {
      const colors = getColors(node.layer);
      elements.push({
        data: {
          id: node.id,
          label: node.label,
          sublabel: `${node.fileCount} file${node.fileCount !== 1 ? "s" : ""}`,
          layerLabel: node.layer,
          layer: node.layer,
          bgColor: colors.bg,
          borderColor: colors.border,
          dotColor: colors.dot,
          textColor: colors.text,
        },
      });
    }

    for (const edge of this.moduleGraph.edges) {
      elements.push({
        data: {
          source: edge.source,
          target: edge.target,
          weight: edge.weight,
        },
      });
    }

    this.cy = cytoscape({
      container: this.container,
      elements,
      style: [
        {
          selector: "node",
          style: {
            shape: "roundrectangle",
            width: 160,
            height: 48,
            "background-color": "data(bgColor)",
            "border-width": 1.5,
            "border-color": "data(borderColor)",
            "border-opacity": 0.6,
            label: "data(label)",
            "font-size": "11px",
            "font-family": "'JetBrains Mono', 'Fira Code', monospace",
            "font-weight": "600" as unknown as number,
            color: "data(textColor)",
            "text-valign": "center",
            "text-halign": "center",
            "text-margin-y": -6,
            "text-outline-width": 0,
          },
        },
        {
          selector: "node:selected",
          style: {
            "border-width": 2,
            "border-color": "#ffffff",
            "border-opacity": 1,
          },
        },
        {
          selector: "node:active",
          style: {
            "overlay-opacity": 0,
          },
        },
        {
          selector: "edge",
          style: {
            width: 1.5,
            "line-color": "#1e293b",
            "target-arrow-color": "#334155",
            "target-arrow-shape": "triangle",
            "arrow-scale": 0.8,
            "curve-style": "bezier",
            opacity: 0.5,
          },
        },
        {
          selector: "edge:selected",
          style: {
            "line-color": "#64748b",
            "target-arrow-color": "#64748b",
            opacity: 1,
          },
        },
      ],
      layout: {
        name: "dagre",
        rankDir: "TB",
        spacingFactor: 1.6,
        nodeSep: 60,
        rankSep: 100,
        animate: false,
      } as cytoscape.LayoutOptions,
      minZoom: 0.3,
      maxZoom: 2.5,
      wheelSensitivity: 0.3,
    });

    // Render HTML card overlays after layout
    this.renderCardOverlays();

    // Update overlay positions on pan/zoom
    this.cy.on("viewport", () => this.updateOverlayPositions());
    this.cy.on("position", () => this.updateOverlayPositions());

    // Click handler
    this.cy.on("tap", "node", (evt) => {
      const nodeId = evt.target.id();
      const moduleNode = this.nodeMap.get(nodeId);
      this.onNodeSelect(nodeId, moduleNode);
    });

    this.cy.on("tap", (evt) => {
      if (evt.target === this.cy) {
        this.onNodeSelect(null);
      }
    });
  }

  private renderCardOverlays(): void {
    if (!this.cy) return;
    this.overlay.innerHTML = "";

    this.cy.nodes().forEach((cyNode) => {
      const id = cyNode.id();
      const moduleNode = this.nodeMap.get(id);
      if (!moduleNode) return;

      const colors = getColors(moduleNode.layer);
      const card = document.createElement("div");
      card.className = "module-card";
      card.dataset.nodeId = id;
      card.style.cssText = `
        position: absolute;
        pointer-events: auto;
        cursor: pointer;
        min-width: 140px;
        padding: 8px 14px;
        border-radius: 10px;
        background: ${colors.bg};
        border: 1.5px solid ${colors.border}88;
        transition: border-color 0.15s, box-shadow 0.15s;
      `;

      // Row 1: dot + label
      const row1 = document.createElement("div");
      row1.style.cssText = "display:flex;align-items:center;gap:7px;margin-bottom:3px;";

      const dot = document.createElement("div");
      dot.style.cssText = `width:7px;height:7px;border-radius:50%;background:${colors.dot};flex-shrink:0;`;
      row1.appendChild(dot);

      const label = document.createElement("span");
      label.style.cssText = `color:${colors.text};font-size:11px;font-family:'JetBrains Mono','Fira Code',monospace;font-weight:600;letter-spacing:0.02em;`;
      label.textContent = moduleNode.label;
      row1.appendChild(label);
      card.appendChild(row1);

      // Row 2: file count + layer badge
      const row2 = document.createElement("div");
      row2.style.cssText = "display:flex;gap:6px;align-items:center;";

      const fileCount = document.createElement("span");
      fileCount.style.cssText = "color:#475569;font-size:10px;font-family:monospace;";
      fileCount.textContent = `${moduleNode.fileCount} file${moduleNode.fileCount !== 1 ? "s" : ""}`;
      row2.appendChild(fileCount);

      const badge = document.createElement("span");
      badge.style.cssText = `color:${colors.border};font-size:9px;font-family:monospace;background:${colors.border}18;padding:1px 5px;border-radius:3px;`;
      badge.textContent = moduleNode.layer;
      row2.appendChild(badge);
      card.appendChild(row2);

      card.addEventListener("click", (e) => {
        e.stopPropagation();
        this.onNodeSelect(id, moduleNode);
      });

      card.addEventListener("mouseenter", () => {
        card.style.borderColor = colors.border;
        card.style.boxShadow = `0 0 0 3px ${colors.border}44, 0 8px 32px #00000080`;
      });
      card.addEventListener("mouseleave", () => {
        card.style.borderColor = `${colors.border}88`;
        card.style.boxShadow = "none";
      });

      this.overlay.appendChild(card);
    });

    this.updateOverlayPositions();
  }

  private updateOverlayPositions(): void {
    if (!this.cy) return;

    const pan = this.cy.pan();
    const zoom = this.cy.zoom();

    this.overlay.querySelectorAll<HTMLElement>(".module-card").forEach((card) => {
      const nodeId = card.dataset.nodeId;
      if (!nodeId) return;

      const cyNode = this.cy!.getElementById(nodeId);
      if (cyNode.length === 0) return;

      const pos = cyNode.position();
      const x = pos.x * zoom + pan.x;
      const y = pos.y * zoom + pan.y;

      // Center the card on the node position
      const w = card.offsetWidth;
      const h = card.offsetHeight;
      card.style.left = `${x - w / 2}px`;
      card.style.top = `${y - h / 2}px`;
      card.style.transform = `scale(${Math.min(1, zoom)})`;
      card.style.transformOrigin = "center center";
    });
  }

  highlightNode(nodeId: string): void {
    if (!this.cy) return;
    this.cy.elements().unselect();
    if (nodeId) {
      const node = this.cy.getElementById(nodeId);
      if (node.length > 0) {
        node.select();
        this.cy.animate({
          center: { eles: node },
          duration: 300,
        });
      }
    }
  }

  getModuleGraph(): ModuleGraph {
    return this.moduleGraph;
  }

  destroy(): void {
    if (this.cy) {
      this.cy.destroy();
      this.cy = null;
    }
  }

  unmount(): void {
    this.destroy();
    this.overlay.remove();
    this.container.remove();
  }

  getContainer(): HTMLElement {
    return this.container;
  }
}
