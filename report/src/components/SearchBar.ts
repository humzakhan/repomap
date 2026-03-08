import Fuse from "fuse.js";
import type { ModuleSummary, SearchEntry } from "../types";

export interface SearchBarOptions {
  summaries: ModuleSummary[];
  onSelect: (filePath: string) => void;
}

export class SearchBar {
  private el: HTMLElement;
  private input: HTMLInputElement;
  private resultsEl: HTMLElement;
  private fuse: Fuse<SearchEntry>;
  private entries: SearchEntry[];
  private activeIndex = -1;
  private results: SearchEntry[] = [];
  private onSelect: (filePath: string) => void;

  constructor(options: SearchBarOptions) {
    this.onSelect = options.onSelect;

    // Build search index
    this.entries = options.summaries.map((s) => ({
      id: s.file_path,
      label: s.file_path.split("/").pop() || s.file_path,
      file_path: s.file_path,
      kind: s.layer,
      keywords: [
        s.summary,
        ...s.responsibilities,
        ...s.exports.map((e) => e.name),
        ...s.patterns,
      ].join(" "),
    }));

    this.fuse = new Fuse(this.entries, {
      keys: [
        { name: "label", weight: 2 },
        { name: "file_path", weight: 1.5 },
        { name: "keywords", weight: 1 },
      ],
      threshold: 0.4,
      includeScore: true,
    });

    // Build DOM
    this.el = document.createElement("div");
    this.el.className = "search-bar";

    const icon = document.createElement("span");
    icon.className = "search-bar__icon";
    icon.textContent = "\u2315"; // search icon
    this.el.appendChild(icon);

    this.input = document.createElement("input");
    this.input.className = "search-bar__input";
    this.input.type = "text";
    this.input.placeholder = "Search files...";
    this.input.addEventListener("input", () => this.handleInput());
    this.input.addEventListener("keydown", (e) => this.handleKeydown(e));
    this.input.addEventListener("focus", () => this.handleInput());
    this.input.addEventListener("blur", () => {
      // Delay to allow click on result
      setTimeout(() => this.hideResults(), 150);
    });
    this.el.appendChild(this.input);

    this.resultsEl = document.createElement("div");
    this.resultsEl.className = "search-bar__results";
    this.resultsEl.style.display = "none";
    this.el.appendChild(this.resultsEl);
  }

  private handleInput(): void {
    const query = this.input.value.trim();
    if (!query) {
      this.hideResults();
      return;
    }

    const fuseResults = this.fuse.search(query, { limit: 10 });
    this.results = fuseResults.map((r) => r.item);
    this.activeIndex = -1;
    this.renderResults();
  }

  private handleKeydown(e: KeyboardEvent): void {
    if (e.key === "ArrowDown") {
      e.preventDefault();
      this.activeIndex = Math.min(this.activeIndex + 1, this.results.length - 1);
      this.renderResults();
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      this.activeIndex = Math.max(this.activeIndex - 1, 0);
      this.renderResults();
    } else if (e.key === "Enter" && this.activeIndex >= 0) {
      e.preventDefault();
      const entry = this.results[this.activeIndex];
      if (entry) {
        this.onSelect(entry.file_path);
        this.input.value = "";
        this.hideResults();
      }
    } else if (e.key === "Escape") {
      this.input.value = "";
      this.hideResults();
      this.input.blur();
    }
  }

  private renderResults(): void {
    this.resultsEl.innerHTML = "";
    if (this.results.length === 0) {
      this.resultsEl.style.display = "none";
      return;
    }
    this.resultsEl.style.display = "block";

    this.results.forEach((entry, i) => {
      const item = document.createElement("div");
      item.className = "search-bar__result";
      if (i === this.activeIndex) {
        item.classList.add("search-bar__result--active");
      }

      const dot = document.createElement("span");
      dot.className = "nav-tree__dot";
      dot.style.backgroundColor = layerColor(entry.kind);
      item.appendChild(dot);

      const label = document.createElement("span");
      label.textContent = entry.label;
      item.appendChild(label);

      const path = document.createElement("span");
      path.className = "search-bar__result-path";
      path.textContent = entry.file_path;
      item.appendChild(path);

      item.addEventListener("click", () => {
        this.onSelect(entry.file_path);
        this.input.value = "";
        this.hideResults();
      });

      this.resultsEl.appendChild(item);
    });
  }

  private hideResults(): void {
    this.resultsEl.style.display = "none";
    this.results = [];
    this.activeIndex = -1;
  }

  getElement(): HTMLElement {
    return this.el;
  }
}

function layerColor(layer: string): string {
  const colors: Record<string, string> = {
    api: "#7c3aed",
    service: "#06b6d4",
    data: "#f59e0b",
    util: "#64748b",
    test: "#10b981",
    config: "#ec4899",
    unknown: "#94a3b8",
  };
  return colors[layer] || colors.unknown;
}
