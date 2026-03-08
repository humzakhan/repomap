import type { CriticalPath } from "../types";

export interface FlowExplorerOptions {
  criticalPaths: CriticalPath[];
  onNodeSelect: (nodeId: string) => void;
}

export class FlowExplorer {
  private container: HTMLElement;
  private paths: CriticalPath[];
  private onNodeSelect: (nodeId: string) => void;

  constructor(options: FlowExplorerOptions) {
    this.paths = options.criticalPaths;
    this.onNodeSelect = options.onNodeSelect;
    this.container = document.createElement("div");
    this.container.className = "canvas__mermaid";
  }

  async mount(parent: HTMLElement): Promise<void> {
    parent.appendChild(this.container);
    await this.renderDiagrams();
  }

  private async renderDiagrams(): Promise<void> {
    if (this.paths.length === 0) {
      const empty = document.createElement("div");
      empty.className = "canvas__empty";
      empty.textContent = "No critical paths available. Run analysis with architecture synthesis enabled.";
      this.container.appendChild(empty);
      return;
    }

    const mermaid = (await import("mermaid")).default;
    mermaid.initialize({
      startOnLoad: false,
      theme: "dark",
      themeVariables: {
        primaryColor: "#7c3aed",
        primaryTextColor: "#f1f5f9",
        primaryBorderColor: "#1e1e2e",
        lineColor: "#64748b",
        secondaryColor: "#111118",
        tertiaryColor: "#1a1a24",
        background: "#0a0a0f",
        mainBkg: "#111118",
        nodeBorder: "#1e1e2e",
        clusterBkg: "#111118",
        titleColor: "#f1f5f9",
        edgeLabelBackground: "#111118",
      },
      sequence: {
        actorMargin: 50,
        noteMargin: 10,
        messageMargin: 30,
      },
    });

    for (let i = 0; i < this.paths.length; i++) {
      const path = this.paths[i];
      const wrapper = document.createElement("div");
      wrapper.className = "mermaid-diagram";

      const title = document.createElement("div");
      title.className = "mermaid-diagram__title";
      title.textContent = path.name;
      wrapper.appendChild(title);

      const desc = document.createElement("div");
      desc.className = "mermaid-diagram__desc";
      desc.textContent = path.description;
      wrapper.appendChild(desc);

      const diagramDef = this.buildSequenceDiagram(path);
      const diagramEl = document.createElement("div");
      diagramEl.className = "mermaid-render";

      try {
        const { svg } = await mermaid.render(`mermaid-${i}`, diagramDef);
        diagramEl.innerHTML = svg;
      } catch {
        diagramEl.textContent = "Failed to render diagram";
        diagramEl.style.color = "#64748b";
      }

      wrapper.appendChild(diagramEl);

      // Add clickable step list below diagram
      const stepList = document.createElement("div");
      stepList.style.marginTop = "12px";
      for (const step of path.steps) {
        const stepEl = document.createElement("div");
        stepEl.className = "detail-panel__dep";
        stepEl.textContent = `${step.module} - ${step.action}: ${step.description}`;
        stepEl.addEventListener("click", () => {
          this.onNodeSelect(step.module);
        });
        stepList.appendChild(stepEl);
      }
      wrapper.appendChild(stepList);

      this.container.appendChild(wrapper);
    }
  }

  private buildSequenceDiagram(path: CriticalPath): string {
    const lines: string[] = ["sequenceDiagram"];

    // Collect unique participants in order
    const seen = new Set<string>();
    const participants: string[] = [];
    for (const step of path.steps) {
      const alias = this.sanitizeAlias(step.module);
      if (!seen.has(alias)) {
        seen.add(alias);
        participants.push(alias);
        const shortName = step.module.split("/").pop() || step.module;
        lines.push(`    participant ${alias} as ${shortName}`);
      }
    }

    // Add arrows between consecutive steps
    for (let i = 0; i < path.steps.length - 1; i++) {
      const from = this.sanitizeAlias(path.steps[i].module);
      const to = this.sanitizeAlias(path.steps[i + 1].module);
      const action = path.steps[i].action.replace(/"/g, "'");
      lines.push(`    ${from}->>+${to}: ${action}`);
    }

    // Add return arrows
    for (let i = path.steps.length - 2; i >= 0; i--) {
      const from = this.sanitizeAlias(path.steps[i + 1].module);
      const to = this.sanitizeAlias(path.steps[i].module);
      lines.push(`    ${from}-->>-${to}: response`);
    }

    return lines.join("\n");
  }

  private sanitizeAlias(module: string): string {
    return module.replace(/[^a-zA-Z0-9]/g, "_");
  }

  unmount(): void {
    this.container.remove();
  }

  getContainer(): HTMLElement {
    return this.container;
  }
}
