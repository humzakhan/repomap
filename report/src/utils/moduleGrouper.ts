import type { ModuleSummary, RepoGraph, GraphEdge, LayerType } from "../types";

export interface ModuleNode {
  id: string;
  label: string;
  files: string[];
  fileCount: number;
  layer: LayerType;
  summary: string;
}

export interface ModuleGraph {
  nodes: ModuleNode[];
  edges: GraphEdge[];
}

const LAYER_HINTS: Record<string, LayerType> = {
  api: "api",
  routes: "api",
  handlers: "api",
  middleware: "api",
  pages: "api",
  service: "service",
  services: "service",
  aiprovider: "service",
  classifier: "service",
  recurrence: "service",
  workers: "service",
  hooks: "service",
  context: "service",
  model: "data",
  models: "data",
  domain: "data",
  repository: "data",
  storage: "data",
  db: "data",
  migrations: "data",
  types: "data",
  config: "config",
  cmd: "api",
  components: "api",
  lib: "util",
  pkg: "util",
  util: "util",
  utils: "util",
  helpers: "util",
  test: "test",
  testutil: "test",
  parser: "service",
  report: "service",
};

/**
 * Groups individual file summaries into module-level nodes.
 * A "module" = a meaningful directory grouping (package, folder of related files).
 */
export function groupIntoModules(
  summaries: ModuleSummary[],
  graph: RepoGraph,
): ModuleGraph {
  // Step 1: Determine module ID for each file
  const fileToModule = new Map<string, string>();
  const moduleFiles = new Map<string, string[]>();

  for (const s of summaries) {
    const moduleId = getModuleId(s.file_path);
    fileToModule.set(s.file_path, moduleId);
    if (!moduleFiles.has(moduleId)) moduleFiles.set(moduleId, []);
    moduleFiles.get(moduleId)!.push(s.file_path);
  }

  // Also map graph node paths (some may not be in summaries)
  for (const node of graph.nodes) {
    const path = node.path || node.id;
    if (!fileToModule.has(path)) {
      const moduleId = getModuleId(path);
      fileToModule.set(path, moduleId);
      if (!moduleFiles.has(moduleId)) moduleFiles.set(moduleId, []);
      moduleFiles.get(moduleId)!.push(path);
    }
  }

  // Step 2: Build module nodes
  const summaryMap = new Map<string, ModuleSummary>();
  for (const s of summaries) summaryMap.set(s.file_path, s);

  const nodes: ModuleNode[] = [];
  for (const [moduleId, files] of moduleFiles) {
    const layer = inferLayer(moduleId, files, summaryMap);
    const summary = buildModuleSummary(files, summaryMap);
    const label = getModuleLabel(moduleId);

    nodes.push({
      id: moduleId,
      label,
      files,
      fileCount: files.length,
      layer,
      summary,
    });
  }

  // Sort: more files first
  nodes.sort((a, b) => b.fileCount - a.fileCount);

  // Step 3: Aggregate edges to module level
  const edgeSet = new Set<string>();
  const edges: GraphEdge[] = [];

  for (const edge of graph.edges) {
    const sourceModule = fileToModule.get(edge.source);
    const targetModule = fileToModule.get(edge.target);
    if (!sourceModule || !targetModule) continue;
    if (sourceModule === targetModule) continue; // skip intra-module

    const key = `${sourceModule}→${targetModule}`;
    if (edgeSet.has(key)) continue;
    edgeSet.add(key);

    edges.push({
      source: sourceModule,
      target: targetModule,
      weight: 1,
    });
  }

  return { nodes, edges };
}

/**
 * Determines the module ID (directory path) for a given file path.
 * Heuristic: use 2-3 levels of directory depth depending on structure.
 */
function getModuleId(filePath: string): string {
  const parts = filePath.split("/");

  // Strip filename — group by directory
  if (parts.length <= 1) return parts[0];

  const dirParts = parts.slice(0, -1);

  // For deep structures like server/internal/api/handlers/foo.go,
  // group at the 3rd level: server/internal/api
  if (dirParts.length >= 3 && (dirParts[1] === "internal" || dirParts[1] === "src")) {
    return dirParts.slice(0, 3).join("/");
  }

  // For web/src/components/Foo.tsx → web/src/components
  if (dirParts.length >= 3) {
    return dirParts.slice(0, 3).join("/");
  }

  // For shallow paths like server/cmd → use full dir
  if (dirParts.length >= 2) {
    return dirParts.join("/");
  }

  return dirParts[0];
}

/**
 * Creates a short display label from a module ID.
 * e.g., "server/internal/api" → "api"
 */
function getModuleLabel(moduleId: string): string {
  const parts = moduleId.split("/");

  // For "server/internal/X" → "X"
  if (parts.length >= 3 && (parts[1] === "internal" || parts[1] === "src")) {
    return parts.slice(2).join("/");
  }

  // For "web/src/components" → "components"
  if (parts.length >= 2) {
    return parts[parts.length - 1];
  }

  return moduleId;
}

function inferLayer(
  moduleId: string,
  files: string[],
  summaryMap: Map<string, ModuleSummary>,
): LayerType {
  // Check directory name against hints
  const parts = moduleId.split("/");
  const dirName = parts[parts.length - 1].toLowerCase();

  if (LAYER_HINTS[dirName]) return LAYER_HINTS[dirName];

  // Check if summaries have a consistent layer
  const layers = new Map<LayerType, number>();
  for (const f of files) {
    const s = summaryMap.get(f);
    if (s && s.layer && s.layer !== "unknown") {
      layers.set(s.layer, (layers.get(s.layer) || 0) + 1);
    }
  }

  if (layers.size > 0) {
    // Return most common non-unknown layer
    let best: LayerType = "unknown";
    let bestCount = 0;
    for (const [layer, count] of layers) {
      if (count > bestCount) {
        best = layer;
        bestCount = count;
      }
    }
    return best;
  }

  return "unknown";
}

function buildModuleSummary(
  files: string[],
  summaryMap: Map<string, ModuleSummary>,
): string {
  // Combine summaries from constituent files
  const summaries: string[] = [];
  for (const f of files) {
    const s = summaryMap.get(f);
    if (s && s.summary) summaries.push(s.summary);
  }

  if (summaries.length === 0) return "";
  if (summaries.length === 1) return summaries[0];

  // Take first 2-3 summaries to keep it concise
  return summaries.slice(0, 3).join(" ");
}
