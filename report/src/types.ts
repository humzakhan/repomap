/** Types matching the Go PipelineResult output */

export interface RepoMapData {
  summaries: ModuleSummary[];
  architecture?: ArchitectureSynthesis;
  doc_warnings?: DocWarning[];
  stats: PipelineStats;
  graph: RepoGraph;
  metadata: RepoMetadata;
}

export interface ModuleSummary {
  file_path: string;
  language: string;
  summary: string;
  responsibilities: string[];
  patterns: string[];
  dependencies_on: string[];
  depended_on_by_hint: string[] | null;
  layer: LayerType;
  exports: ExportedSymbol[];
  imports: ImportedModule[];
  source_code?: string;
  token_count: number;
  status: "complete" | "summary_unavailable";
}

export type LayerType =
  | "api"
  | "service"
  | "data"
  | "util"
  | "test"
  | "config"
  | "unknown";

export interface ExportedSymbol {
  name: string;
  kind: "function" | "class" | "interface" | "type" | "variable" | "constant";
  line: number;
}

export interface ImportedModule {
  path: string;
  symbols: string[];
  is_external: boolean;
}

export interface ArchitectureSynthesis {
  narrative: string;
  layers: LayerDefinition[];
  critical_paths: CriticalPath[];
  start_here: string[];
  design_patterns: string[];
}

export interface LayerDefinition {
  name: string;
  type: LayerType;
  modules: string[];
  description: string;
}

export interface CriticalPath {
  name: string;
  description: string;
  steps: FlowStep[];
}

export interface FlowStep {
  module: string;
  action: string;
  description: string;
}

export interface DocWarning {
  file_path: string;
  warning: string;
  severity: "info" | "warning" | "error";
}

export interface PipelineStats {
  total_files: number;
  total_modules: number;
  total_tokens_used: number;
  total_cost: number;
  model_used: string;
  duration_ms: number;
  languages: LanguageStat[];
}

export interface LanguageStat {
  language: string;
  file_count: number;
  percentage: number;
}

export interface RepoGraph {
  nodes: GraphNode[];
  edges: GraphEdge[];
}

export interface GraphNode {
  id: string;
  label: string;
  file_path: string;
  layer: LayerType;
  size: number;
}

export interface GraphEdge {
  source: string;
  target: string;
  weight: number;
  label?: string;
}

export interface RepoMetadata {
  repo_name: string;
  generated_at: string;
  model: string;
  commit_hash: string;
  branch: string;
}

/** Search index entry for Fuse.js */
export interface SearchEntry {
  id: string;
  label: string;
  file_path: string;
  kind: string;
  keywords: string;
}
