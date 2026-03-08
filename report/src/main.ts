import "./styles/main.css";
import type { RepoMapData, ModuleSummary } from "./types";
import { Toolbar, type ViewName } from "./components/Toolbar";
import { SearchBar } from "./components/SearchBar";
import { NavTree } from "./components/NavTree";
import { DetailPanel } from "./components/DetailPanel";
import { StatusBar } from "./components/StatusBar";
import { ArchitectureMap } from "./views/ArchitectureMap";
import { FlowExplorer } from "./views/FlowExplorer";
import { ModelCanvas } from "./views/ModelCanvas";

// Load data from embedded JSON or fixture
async function loadData(): Promise<RepoMapData> {
  const scriptTag = document.getElementById("repomap-data");
  if (scriptTag && scriptTag.textContent && scriptTag.textContent.trim().length > 2) {
    return JSON.parse(scriptTag.textContent);
  }
  // Dev mode: load fixture
  const fixture = await import("./fixtures/sample.json");
  return fixture.default as unknown as RepoMapData;
}

class App {
  private data!: RepoMapData;
  private summaryMap: Map<string, ModuleSummary> = new Map();

  // Components
  private toolbar!: Toolbar;
  private searchBar!: SearchBar;
  private navTree!: NavTree;
  private detailPanel!: DetailPanel;
  private statusBar!: StatusBar;

  // Views
  private architectureMap: ArchitectureMap | null = null;
  private flowExplorer: FlowExplorer | null = null;
  private modelCanvas: ModelCanvas | null = null;

  private canvasEl!: HTMLElement;
  private currentView: ViewName = "architecture";

  async init(): Promise<void> {
    this.data = await loadData();

    // Build lookup map
    for (const s of this.data.summaries) {
      this.summaryMap.set(s.file_path, s);
    }

    const app = document.getElementById("app")!;
    app.innerHTML = "";

    // Toolbar
    this.toolbar = new Toolbar({
      repoName: this.data.metadata.repo_name,
      onViewChange: (view) => this.switchView(view),
    });
    this.toolbar.mount(app);

    // Search bar (appended to toolbar)
    this.searchBar = new SearchBar({
      summaries: this.data.summaries,
      onSelect: (fp) => this.selectModule(fp),
    });
    this.toolbar.appendRight(this.searchBar.getElement());

    // Nav tree
    this.navTree = new NavTree({
      summaries: this.data.summaries,
      onSelect: (fp) => this.selectModule(fp),
    });
    this.navTree.mount(app);

    // Canvas (views mount here)
    this.canvasEl = document.createElement("div");
    this.canvasEl.className = "canvas";
    app.appendChild(this.canvasEl);

    // Detail panel
    this.detailPanel = new DetailPanel({
      onNavigate: (fp) => this.selectModule(fp),
    });
    this.detailPanel.mount(app);

    // Status bar
    this.statusBar = new StatusBar({
      stats: this.data.stats,
      graph: this.data.graph,
    });
    this.statusBar.mount(app);

    // Initial view
    this.switchView("architecture");
  }

  private switchView(view: ViewName): void {
    // Unmount current view
    this.unmountCurrentView();
    this.currentView = view;

    switch (view) {
      case "architecture":
        this.architectureMap = new ArchitectureMap({
          graph: this.data.graph,
          onNodeSelect: (id) => {
            if (id) {
              this.selectModule(id);
            } else {
              this.detailPanel.showEmpty();
            }
          },
        });
        this.architectureMap.mount(this.canvasEl);
        break;

      case "flows":
        this.flowExplorer = new FlowExplorer({
          criticalPaths: this.data.architecture?.critical_paths || [],
          onNodeSelect: (id) => this.selectModule(id),
        });
        this.flowExplorer.mount(this.canvasEl);
        break;

      case "models":
        this.modelCanvas = new ModelCanvas({
          stats: this.data.stats,
          graph: this.data.graph,
          summaries: this.data.summaries,
          onNodeSelect: (id) => this.selectModule(id),
        });
        this.modelCanvas.mount(this.canvasEl);
        break;
    }
  }

  private unmountCurrentView(): void {
    if (this.architectureMap) {
      this.architectureMap.unmount();
      this.architectureMap = null;
    }
    if (this.flowExplorer) {
      this.flowExplorer.unmount();
      this.flowExplorer = null;
    }
    if (this.modelCanvas) {
      this.modelCanvas.unmount();
      this.modelCanvas = null;
    }
  }

  private selectModule(filePath: string): void {
    const summary = this.summaryMap.get(filePath);
    if (!summary) return;

    this.navTree.select(filePath);
    this.detailPanel.show(summary);

    // If on architecture view, highlight the node
    if (this.architectureMap) {
      this.architectureMap.highlightNode(filePath);
    }
  }
}

// Boot
const app = new App();
app.init().catch((err) => {
  console.error("Failed to initialize Repomap report:", err);
  const appEl = document.getElementById("app");
  if (appEl) {
    appEl.innerHTML = `<div style="padding:32px;color:#f1f5f9;">
      <h2>Failed to load report</h2>
      <pre style="color:#64748b;">${String(err)}</pre>
    </div>`;
  }
});
