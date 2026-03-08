import cytoscape from "cytoscape";
import cytoscapeDagre from "cytoscape-dagre";
import type { RepoGraph, LayerType } from "../types";

// Register dagre layout
cytoscapeDagre(cytoscape);

const LAYER_COLORS: Record<LayerType | "unknown", string> = {
  api: "#7c3aed",
  service: "#06b6d4",
  data: "#f59e0b",
  util: "#64748b",
  test: "#10b981",
  config: "#ec4899",
  unknown: "#94a3b8",
};

export interface ArchitectureMapOptions {
  graph: RepoGraph;
  onNodeSelect: (nodeId: string) => void;
}

export class ArchitectureMap {
  private cy: cytoscape.Core | null = null;
  private container: HTMLElement;
  private onNodeSelect: (nodeId: string) => void;
  private graph: RepoGraph;

  constructor(options: ArchitectureMapOptions) {
    this.graph = options.graph;
    this.onNodeSelect = options.onNodeSelect;
    this.container = document.createElement("div");
    this.container.className = "canvas__cytoscape";
  }

  mount(parent: HTMLElement): void {
    parent.appendChild(this.container);
    this.initCytoscape();
  }

  private initCytoscape(): void {
    const elements: cytoscape.ElementDefinition[] = [];

    // Add nodes
    for (const node of this.graph.nodes) {
      const color = LAYER_COLORS[node.layer] || LAYER_COLORS.unknown;
      elements.push({
        data: {
          id: node.id,
          label: node.label,
          layer: node.layer,
          color,
          size: Math.max(20, Math.min(60, Math.sqrt(node.size) * 1.5)),
        },
      });
    }

    // Add edges
    for (const edge of this.graph.edges) {
      elements.push({
        data: {
          source: edge.source,
          target: edge.target,
          weight: edge.weight,
          label: edge.label || "",
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
            label: "data(label)",
            "background-color": "data(color)",
            width: "data(size)",
            height: "data(size)",
            "font-size": "10px",
            color: "#f1f5f9",
            "text-valign": "bottom",
            "text-margin-y": 6,
            "border-width": 2,
            "border-color": "data(color)",
            "border-opacity": 0.3,
            "text-outline-width": 2,
            "text-outline-color": "#0a0a0f",
            "text-outline-opacity": 0.8,
          },
        },
        {
          selector: "node:selected",
          style: {
            "border-width": 3,
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
            width: "mapData(weight, 1, 5, 1, 3)",
            "line-color": "#1e1e2e",
            "target-arrow-color": "#1e1e2e",
            "target-arrow-shape": "triangle",
            "curve-style": "bezier",
            opacity: 0.6,
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
        spacingFactor: 1.4,
        nodeSep: 50,
        rankSep: 80,
        animate: false,
      } as cytoscape.LayoutOptions,
      minZoom: 0.2,
      maxZoom: 3,
      wheelSensitivity: 0.3,
    });

    // Click handler
    this.cy.on("tap", "node", (evt) => {
      const nodeId = evt.target.id();
      this.onNodeSelect(nodeId);
    });

    // Tap on background deselects
    this.cy.on("tap", (evt) => {
      if (evt.target === this.cy) {
        this.onNodeSelect("");
      }
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

  destroy(): void {
    if (this.cy) {
      this.cy.destroy();
      this.cy = null;
    }
  }

  unmount(): void {
    this.destroy();
    this.container.remove();
  }

  getContainer(): HTMLElement {
    return this.container;
  }
}
