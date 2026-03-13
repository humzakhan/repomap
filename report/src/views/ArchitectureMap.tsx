import React, { useEffect, useRef, useCallback } from "react";
import cytoscape from "cytoscape";
import cytoscapeDagre from "cytoscape-dagre";
import type { RepoGraph, ModuleSummary } from "../types";
import { groupIntoModules, type ModuleNode, type ModuleGraph } from "../utils/moduleGrouper";
import { getLayerTheme } from "../utils/layerColors";

cytoscapeDagre(cytoscape);

interface ArchitectureMapProps {
  graph: RepoGraph;
  summaries: ModuleSummary[];
  onNodeSelect: (moduleId: string | null, moduleNode?: ModuleNode) => void;
}

export function ArchitectureMap({ graph, summaries, onNodeSelect }: ArchitectureMapProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const overlayRef = useRef<HTMLDivElement>(null);
  const cyRef = useRef<cytoscape.Core | null>(null);
  const moduleGraphRef = useRef<ModuleGraph | null>(null);
  const nodeMapRef = useRef<Map<string, ModuleNode>>(new Map());

  const updateOverlayPositions = useCallback(() => {
    const cy = cyRef.current;
    const overlay = overlayRef.current;
    if (!cy || !overlay) return;

    const pan = cy.pan();
    const zoom = cy.zoom();

    overlay.querySelectorAll<HTMLElement>(".module-card").forEach((card) => {
      const nodeId = card.dataset.nodeId;
      if (!nodeId) return;

      const cyNode = cy.getElementById(nodeId);
      if (cyNode.length === 0) return;

      const pos = cyNode.position();
      const x = pos.x * zoom + pan.x;
      const y = pos.y * zoom + pan.y;

      const w = card.offsetWidth;
      const h = card.offsetHeight;
      card.style.left = `${x - w / 2}px`;
      card.style.top = `${y - h / 2}px`;
      card.style.transform = `scale(${Math.min(1, zoom)})`;
      card.style.transformOrigin = "center center";
    });
  }, []);

  useEffect(() => {
    if (!containerRef.current || !overlayRef.current) return;

    const moduleGraph = groupIntoModules(summaries, graph);
    moduleGraphRef.current = moduleGraph;
    const nodeMap = new Map<string, ModuleNode>();
    for (const n of moduleGraph.nodes) nodeMap.set(n.id, n);
    nodeMapRef.current = nodeMap;

    const elements: cytoscape.ElementDefinition[] = [];

    for (const node of moduleGraph.nodes) {
      const colors = getLayerTheme(node.layer);
      elements.push({
        data: {
          id: node.id,
          label: node.label,
          layer: node.layer,
          bgColor: colors.bg,
          borderColor: colors.border,
          textColor: colors.text,
        },
      });
    }

    for (const edge of moduleGraph.edges) {
      elements.push({
        data: { source: edge.source, target: edge.target, weight: edge.weight },
      });
    }

    const cy = cytoscape({
      container: containerRef.current,
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
          style: { "border-width": 2, "border-color": "#ffffff", "border-opacity": 1 },
        },
        {
          selector: "node:active",
          style: { "overlay-opacity": 0 },
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
          style: { "line-color": "#64748b", "target-arrow-color": "#64748b", opacity: 1 },
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

    cyRef.current = cy;

    // Render card overlays
    renderCardOverlays(overlayRef.current, cy, nodeMap, onNodeSelect);
    updateOverlayPositions();

    cy.on("viewport", updateOverlayPositions);
    cy.on("position", updateOverlayPositions);

    cy.on("tap", "node", (evt) => {
      const nodeId = evt.target.id();
      onNodeSelect(nodeId, nodeMap.get(nodeId));
    });

    cy.on("tap", (evt) => {
      if (evt.target === cy) onNodeSelect(null);
    });

    return () => {
      cy.destroy();
      cyRef.current = null;
    };
  }, [graph, summaries, onNodeSelect, updateOverlayPositions]);

  return (
    <div style={{ position: "relative", width: "100%", height: "100%" }}>
      <div ref={containerRef} style={{ width: "100%", height: "100%" }} />
      <div
        ref={overlayRef}
        style={{
          position: "absolute",
          inset: 0,
          pointerEvents: "none",
          overflow: "hidden",
        }}
      />
    </div>
  );
}

function renderCardOverlays(
  overlay: HTMLElement,
  cy: cytoscape.Core,
  nodeMap: Map<string, ModuleNode>,
  onNodeSelect: (id: string, node?: ModuleNode) => void,
) {
  overlay.innerHTML = "";

  cy.nodes().forEach((cyNode) => {
    const id = cyNode.id();
    const moduleNode = nodeMap.get(id);
    if (!moduleNode) return;

    const colors = getLayerTheme(moduleNode.layer);
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
      onNodeSelect(id, moduleNode);
    });

    card.addEventListener("mouseenter", () => {
      card.style.borderColor = colors.border;
      card.style.boxShadow = `0 0 0 3px ${colors.border}44, 0 8px 32px #00000080`;
    });
    card.addEventListener("mouseleave", () => {
      card.style.borderColor = `${colors.border}88`;
      card.style.boxShadow = "none";
    });

    overlay.appendChild(card);
  });
}
