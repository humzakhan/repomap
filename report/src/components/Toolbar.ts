export type ViewName = "architecture";

export interface ToolbarOptions {
  repoName: string;
  onViewChange: (view: ViewName) => void;
}

const TABS: { id: ViewName; label: string }[] = [
  { id: "architecture", label: "Architecture" },
];

export class Toolbar {
  private el: HTMLElement;
  private activeView: ViewName = "architecture";
  private tabButtons: Map<ViewName, HTMLButtonElement> = new Map();
  private onViewChange: (view: ViewName) => void;
  private mounted = false;

  constructor(options: ToolbarOptions) {
    this.onViewChange = options.onViewChange;
    this.el = document.createElement("div");
    this.el.className = "toolbar";

    // Logo
    const logo = document.createElement("div");
    logo.className = "toolbar__logo";
    logo.textContent = "repomap";
    this.el.appendChild(logo);

    // Repo name
    const repoName = document.createElement("span");
    repoName.className = "toolbar__repo-name";
    repoName.textContent = options.repoName;
    this.el.appendChild(repoName);

    // Tabs
    const tabs = document.createElement("div");
    tabs.className = "toolbar__tabs";

    for (const tab of TABS) {
      const btn = document.createElement("button");
      btn.className = "toolbar__tab";
      btn.textContent = tab.label;
      btn.addEventListener("click", () => this.setActive(tab.id));
      tabs.appendChild(btn);
      this.tabButtons.set(tab.id, btn);
    }

    this.el.appendChild(tabs);

    // Spacer
    const spacer = document.createElement("div");
    spacer.className = "toolbar__spacer";
    this.el.appendChild(spacer);

    this.setActive("architecture");
  }

  setActive(view: ViewName): void {
    this.activeView = view;
    for (const [id, btn] of this.tabButtons) {
      btn.classList.toggle("toolbar__tab--active", id === view);
    }
    if (this.mounted) {
      this.onViewChange(view);
    }
  }

  getActiveView(): ViewName {
    return this.activeView;
  }

  /** Insert the search bar element into the toolbar (right side). */
  appendRight(element: HTMLElement): void {
    this.el.appendChild(element);
  }

  mount(parent: HTMLElement): void {
    parent.appendChild(this.el);
    this.mounted = true;
  }
}
