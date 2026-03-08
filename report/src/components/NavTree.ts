import type { ModuleSummary, LayerType } from "../types";

export interface NavTreeOptions {
  summaries: ModuleSummary[];
  onSelect: (filePath: string) => void;
}

interface TreeGroup {
  name: string;
  layer: LayerType;
  items: ModuleSummary[];
  collapsed: boolean;
}

const LAYER_COLORS: Record<string, string> = {
  api: "#7c3aed",
  service: "#06b6d4",
  data: "#f59e0b",
  util: "#64748b",
  test: "#10b981",
  config: "#ec4899",
  unknown: "#94a3b8",
};

const LAYER_ORDER: LayerType[] = [
  "api",
  "service",
  "data",
  "util",
  "config",
  "test",
  "unknown",
];

export class NavTree {
  private el: HTMLElement;
  private groups: TreeGroup[];
  private selectedPath: string | null = null;
  private onSelect: (filePath: string) => void;

  constructor(options: NavTreeOptions) {
    this.onSelect = options.onSelect;
    this.el = document.createElement("div");
    this.el.className = "nav-tree";

    // Group summaries by layer
    const grouped = new Map<LayerType, ModuleSummary[]>();
    for (const s of options.summaries) {
      const layer = s.layer || "unknown";
      if (!grouped.has(layer)) grouped.set(layer, []);
      grouped.get(layer)!.push(s);
    }

    this.groups = LAYER_ORDER.filter((l) => grouped.has(l)).map((layer) => ({
      name: layer.charAt(0).toUpperCase() + layer.slice(1),
      layer,
      items: grouped.get(layer)!.sort((a, b) =>
        a.file_path.localeCompare(b.file_path)
      ),
      collapsed: false,
    }));

    this.render();
  }

  private render(): void {
    this.el.innerHTML = "";

    for (const group of this.groups) {
      const groupEl = document.createElement("div");
      groupEl.className = "nav-tree__group";

      // Group header
      const header = document.createElement("div");
      header.className = "nav-tree__group-header";

      const chevron = document.createElement("span");
      chevron.className = "nav-tree__chevron";
      if (group.collapsed) chevron.classList.add("nav-tree__chevron--collapsed");
      chevron.textContent = "\u25BE"; // down triangle
      header.appendChild(chevron);

      const label = document.createElement("span");
      label.textContent = `${group.name} (${group.items.length})`;
      header.appendChild(label);

      header.addEventListener("click", () => {
        group.collapsed = !group.collapsed;
        this.render();
      });

      groupEl.appendChild(header);

      // Items
      if (!group.collapsed) {
        for (const item of group.items) {
          const itemEl = document.createElement("div");
          itemEl.className = "nav-tree__item";
          if (item.file_path === this.selectedPath) {
            itemEl.classList.add("nav-tree__item--selected");
          }

          const dot = document.createElement("span");
          dot.className = "nav-tree__dot";
          dot.style.backgroundColor = LAYER_COLORS[group.layer] || LAYER_COLORS.unknown;
          itemEl.appendChild(dot);

          const name = document.createElement("span");
          name.textContent = item.file_path.split("/").pop() || item.file_path;
          name.title = item.file_path;
          itemEl.appendChild(name);

          itemEl.addEventListener("click", () => {
            this.select(item.file_path);
          });

          groupEl.appendChild(itemEl);
        }
      }

      this.el.appendChild(groupEl);
    }
  }

  select(filePath: string): void {
    this.selectedPath = filePath;
    this.render();
    this.onSelect(filePath);
  }

  mount(parent: HTMLElement): void {
    parent.appendChild(this.el);
  }
}
