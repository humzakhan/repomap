import type { LayerType } from "../types";

export const LAYER_COLORS: Record<string, string> = {
  api: "#7c3aed",
  service: "#06b6d4",
  data: "#f59e0b",
  util: "#64748b",
  test: "#10b981",
  config: "#ec4899",
  unknown: "#94a3b8",
};

export const LAYER_THEME: Record<
  string,
  { bg: string; border: string; dot: string; text: string; badge: string }
> = {
  api: { bg: "#1e1040", border: "#7c3aed", dot: "#a78bfa", text: "#c4b5fd", badge: "#7c3aed" },
  service: { bg: "#0e2a33", border: "#0891b2", dot: "#22d3ee", text: "#67e8f9", badge: "#0891b2" },
  data: { bg: "#2a1a00", border: "#d97706", dot: "#fbbf24", text: "#fcd34d", badge: "#d97706" },
  config: { bg: "#1a0a2e", border: "#db2777", dot: "#f472b6", text: "#fbcfe8", badge: "#db2777" },
  util: { bg: "#1a1a24", border: "#64748b", dot: "#94a3b8", text: "#cbd5e1", badge: "#64748b" },
  test: { bg: "#0a2a1a", border: "#059669", dot: "#34d399", text: "#6ee7b7", badge: "#059669" },
  unknown: { bg: "#1a1a24", border: "#475569", dot: "#94a3b8", text: "#cbd5e1", badge: "#475569" },
};

export function getLayerColor(layer: string): string {
  return LAYER_COLORS[layer] || LAYER_COLORS.unknown;
}

export function getLayerTheme(layer: string) {
  return LAYER_THEME[layer] || LAYER_THEME.unknown;
}

export const LAYER_HINTS: Record<string, LayerType> = {
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
  helpers: "util",
  test: "test",
  testutil: "test",
};
