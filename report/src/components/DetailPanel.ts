import type { ModuleSummary } from "../types";

const LAYER_COLORS: Record<string, string> = {
  api: "#7c3aed",
  service: "#06b6d4",
  data: "#f59e0b",
  util: "#64748b",
  test: "#10b981",
  config: "#ec4899",
  unknown: "#94a3b8",
};

export interface DetailPanelOptions {
  onNavigate: (filePath: string) => void;
}

export class DetailPanel {
  private el: HTMLElement;
  private onNavigate: (filePath: string) => void;
  private highlighterReady: Promise<void>;
  private highlighter: {
    codeToHtml: (code: string, opts: { lang: string; theme: string }) => string;
  } | null = null;

  constructor(options: DetailPanelOptions) {
    this.onNavigate = options.onNavigate;
    this.el = document.createElement("div");
    this.el.className = "detail-panel";

    this.highlighterReady = this.initHighlighter();

    this.showEmpty();
  }

  private async initHighlighter(): Promise<void> {
    try {
      const shiki = await import("shiki");
      this.highlighter = await shiki.createHighlighter({
        themes: ["vitesse-dark"],
        langs: ["typescript", "javascript", "python", "go", "rust", "java", "ruby"],
      });
    } catch {
      // Shiki may fail to load in some environments; fall back to plain text
      this.highlighter = null;
    }
  }

  showEmpty(): void {
    this.el.innerHTML = "";
    const empty = document.createElement("div");
    empty.className = "detail-panel__empty";
    empty.textContent = "Select a module from the graph or nav tree to view details";
    this.el.appendChild(empty);
  }

  async show(summary: ModuleSummary): Promise<void> {
    this.el.innerHTML = "";

    // Header
    const header = document.createElement("div");
    header.className = "detail-panel__header";

    const badge = document.createElement("span");
    badge.className = "detail-panel__badge";
    badge.textContent = summary.layer;
    badge.style.backgroundColor = LAYER_COLORS[summary.layer] || LAYER_COLORS.unknown;
    badge.style.color = "#fff";
    header.appendChild(badge);

    const title = document.createElement("div");
    title.className = "detail-panel__title";
    title.textContent = summary.file_path;
    header.appendChild(title);
    this.el.appendChild(header);

    // Summary
    this.addSection("Summary", () => {
      const p = document.createElement("div");
      p.className = "detail-panel__summary";
      p.textContent = summary.summary;
      return p;
    });

    // Responsibilities
    if (summary.responsibilities.length > 0) {
      this.addSection("Responsibilities", () => {
        const ul = document.createElement("ul");
        ul.className = "detail-panel__list";
        for (const r of summary.responsibilities) {
          const li = document.createElement("li");
          li.textContent = r;
          ul.appendChild(li);
        }
        return ul;
      });
    }

    // Patterns
    if (summary.patterns.length > 0) {
      this.addSection("Patterns", () => {
        const ul = document.createElement("ul");
        ul.className = "detail-panel__list";
        for (const p of summary.patterns) {
          const li = document.createElement("li");
          li.textContent = p;
          ul.appendChild(li);
        }
        return ul;
      });
    }

    // Exports
    if (summary.exports.length > 0) {
      this.addSection("Exports", () => {
        const ul = document.createElement("ul");
        ul.className = "detail-panel__list";
        for (const exp of summary.exports) {
          const li = document.createElement("li");
          li.textContent = `${exp.name} (${exp.kind}, line ${exp.line})`;
          ul.appendChild(li);
        }
        return ul;
      });
    }

    // Dependencies (clickable)
    if (summary.dependencies_on.length > 0) {
      this.addSection("Depends On", () => {
        const container = document.createElement("div");
        for (const dep of summary.dependencies_on) {
          const link = document.createElement("a");
          link.className = "detail-panel__dep";
          link.textContent = dep;
          link.href = "#";
          link.addEventListener("click", (e) => {
            e.preventDefault();
            this.onNavigate(dep);
          });
          container.appendChild(link);
        }
        return container;
      });
    }

    // Depended on by
    if (summary.depended_on_by_hint && summary.depended_on_by_hint.length > 0) {
      this.addSection("Depended On By", () => {
        const container = document.createElement("div");
        for (const dep of summary.depended_on_by_hint!) {
          const link = document.createElement("a");
          link.className = "detail-panel__dep";
          link.textContent = dep;
          link.href = "#";
          link.addEventListener("click", (e) => {
            e.preventDefault();
            this.onNavigate(dep);
          });
          container.appendChild(link);
        }
        return container;
      });
    }

    // Source code
    if (summary.source_code) {
      this.addSection("Source Code", () => {
        const codeBlock = document.createElement("div");
        codeBlock.className = "detail-panel__code";

        // Try to highlight with shiki
        if (this.highlighter) {
          try {
            const lang = this.mapLanguage(summary.language);
            codeBlock.innerHTML = this.highlighter.codeToHtml(summary.source_code!, {
              lang,
              theme: "vitesse-dark",
            });
          } catch {
            const pre = document.createElement("pre");
            pre.textContent = summary.source_code!;
            codeBlock.appendChild(pre);
          }
        } else {
          const pre = document.createElement("pre");
          pre.textContent = summary.source_code!;
          codeBlock.appendChild(pre);
        }
        return codeBlock;
      });

      // If highlighter not ready yet, try again once it loads
      if (!this.highlighter) {
        this.highlighterReady.then(() => {
          if (this.highlighter && summary.source_code) {
            const codeBlocks = this.el.querySelectorAll(".detail-panel__code");
            const lastBlock = codeBlocks[codeBlocks.length - 1] as HTMLElement;
            if (lastBlock) {
              try {
                const lang = this.mapLanguage(summary.language);
                lastBlock.innerHTML = this.highlighter.codeToHtml(
                  summary.source_code,
                  { lang, theme: "vitesse-dark" }
                );
              } catch {
                // keep plain text
              }
            }
          }
        });
      }
    }
  }

  private addSection(title: string, buildContent: () => HTMLElement): void {
    const section = document.createElement("div");
    section.className = "detail-panel__section";

    const heading = document.createElement("div");
    heading.className = "detail-panel__section-title";
    heading.textContent = title;
    section.appendChild(heading);

    section.appendChild(buildContent());
    this.el.appendChild(section);
  }

  private mapLanguage(lang: string): string {
    const map: Record<string, string> = {
      typescript: "typescript",
      javascript: "javascript",
      python: "python",
      go: "go",
      rust: "rust",
      java: "java",
      ruby: "ruby",
    };
    return map[lang.toLowerCase()] || "typescript";
  }

  mount(parent: HTMLElement): void {
    parent.appendChild(this.el);
  }
}
