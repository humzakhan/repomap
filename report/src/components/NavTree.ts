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

const LAYER_HINTS: Record<string, LayerType> = {
  api: "api",
  routes: "api",
  handlers: "api",
  middleware: "api",
  pages: "api",
  cmd: "api",
  components: "api",
  service: "service",
  services: "service",
  aiprovider: "service",
  classifier: "service",
  workers: "service",
  hooks: "service",
  context: "service",
  parser: "service",
  report: "service",
  recurrence: "service",
  model: "data",
  models: "data",
  domain: "data",
  repository: "data",
  storage: "data",
  db: "data",
  migrations: "data",
  types: "data",
  config: "config",
  lib: "util",
  pkg: "util",
  util: "util",
  utils: "util",
  test: "test",
  testutil: "test",
};

export class NavTree {
  private el: HTMLElement;
  private groups: TreeGroup[];
  private selectedPath: string | null = null;
  private onSelect: (filePath: string) => void;

  constructor(options: NavTreeOptions) {
    this.onSelect = options.onSelect;
    this.el = document.createElement("div");
    this.el.className = "nav-tree";

    // Group summaries by module directory
    const moduleGroups = new Map<string, ModuleSummary[]>();
    for (const s of options.summaries) {
      const moduleId = this.getModuleDir(s.file_path);
      if (!moduleGroups.has(moduleId)) moduleGroups.set(moduleId, []);
      moduleGroups.get(moduleId)!.push(s);
    }

    this.groups = Array.from(moduleGroups.entries())
      .map(([moduleId, items]) => {
        const dirName = moduleId.split("/").pop() || moduleId;
        const layer = LAYER_HINTS[dirName.toLowerCase()] || "unknown";
        return {
          name: this.getModuleLabel(moduleId),
          layer,
          items: items.sort((a, b) => a.file_path.localeCompare(b.file_path)),
          collapsed: true,
        };
      })
      .sort((a, b) => a.name.localeCompare(b.name));

    this.render();
  }

  private getModuleDir(filePath: string): string {
    const parts = filePath.split("/");
    if (parts.length <= 1) return parts[0];
    const dirParts = parts.slice(0, -1);
    if (dirParts.length >= 3 && (dirParts[1] === "internal" || dirParts[1] === "src")) {
      return dirParts.slice(0, 3).join("/");
    }
    if (dirParts.length >= 3) return dirParts.slice(0, 3).join("/");
    if (dirParts.length >= 2) return dirParts.join("/");
    return dirParts[0];
  }

  private getModuleLabel(moduleId: string): string {
    const parts = moduleId.split("/");
    if (parts.length >= 3 && (parts[1] === "internal" || parts[1] === "src")) {
      return parts.slice(2).join("/");
    }
    if (parts.length >= 2) return parts[parts.length - 1];
    return moduleId;
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

      const dot = document.createElement("span");
      dot.className = "nav-tree__dot";
      dot.style.backgroundColor = LAYER_COLORS[group.layer] || LAYER_COLORS.unknown;
      header.appendChild(dot);

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

          const itemDot = document.createElement("span");
          itemDot.className = "nav-tree__dot";
          itemDot.style.backgroundColor = LAYER_COLORS[group.layer] || LAYER_COLORS.unknown;
          itemEl.appendChild(itemDot);

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
