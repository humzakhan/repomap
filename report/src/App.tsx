import React, { useState, useCallback } from "react";
import { AnimatePresence, motion } from "framer-motion";
import { PanelGroup, Panel, PanelResizeHandle } from "react-resizable-panels";
import { Toolbar, type ViewName } from "./components/Toolbar";
import { SearchBar } from "./components/SearchBar";
import { NavTree } from "./components/NavTree";
import { DetailPanel } from "./components/DetailPanel";
import { StatusBar } from "./components/StatusBar";
import { ArchitectureMap } from "./views/ArchitectureMap";
import { DependencyGraph } from "./views/DependencyGraph";
import { DataModels } from "./views/DataModels";
import { CallFlow } from "./views/CallFlow";
import { useReportData } from "./hooks/useReportData";
import type { ModuleSummary } from "./types";
import type { ModuleNode } from "./utils/moduleGrouper";

export function App() {
  const { data, summaryMap, loading, error } = useReportData();
  const [activeView, setActiveView] = useState<ViewName>("architecture");
  const [selectedPath, setSelectedPath] = useState<string | null>(null);
  const [selection, setSelection] = useState<
    | { type: "file"; summary: ModuleSummary }
    | { type: "module"; moduleNode: ModuleNode; summaryMap: Map<string, ModuleSummary> }
    | null
  >(null);

  const selectFile = useCallback(
    (filePath: string) => {
      const summary = summaryMap.get(filePath);
      if (!summary) return;
      setSelectedPath(filePath);
      setSelection({ type: "file", summary });
    },
    [summaryMap],
  );

  const selectModule = useCallback(
    (moduleId: string | null, moduleNode?: ModuleNode) => {
      if (!moduleNode) {
        setSelection(null);
        return;
      }
      setSelection({ type: "module", moduleNode, summaryMap });
    },
    [summaryMap],
  );

  if (loading) {
    return (
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          height: "100vh",
          color: "#64748b",
        }}
      >
        Loading analysis data...
      </div>
    );
  }

  if (error || !data) {
    return (
      <div style={{ padding: 32, color: "#f1f5f9" }}>
        <h2>Failed to load report</h2>
        <pre style={{ color: "#64748b" }}>{error}</pre>
      </div>
    );
  }

  return (
    <div
      style={{
        display: "grid",
        gridTemplateRows: "auto 1fr auto",
        gridTemplateColumns: "1fr",
        height: "100vh",
        width: "100vw",
      }}
    >
      <Toolbar repoName={data.metadata.repo_name} activeView={activeView} onViewChange={setActiveView}>
        <SearchBar summaries={data.summaries} onSelect={selectFile} />
      </Toolbar>

      <PanelGroup direction="horizontal">
        <Panel defaultSize={18} minSize={12} maxSize={30}>
          <NavTree summaries={data.summaries} selectedPath={selectedPath} onSelect={selectFile} />
        </Panel>
        <PanelResizeHandle
          style={{ width: 1, background: "var(--border)", cursor: "col-resize" }}
        />
        <Panel>
          <PanelGroup direction="vertical">
            <Panel>
              <div className="canvas">
                <AnimatePresence mode="wait">
                  <motion.div
                    key={activeView}
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    exit={{ opacity: 0 }}
                    transition={{ duration: 0.15 }}
                    style={{ width: "100%", height: "100%" }}
                  >
                    {renderView(activeView, data, selectModule, selectFile)}
                  </motion.div>
                </AnimatePresence>
              </div>
            </Panel>
          </PanelGroup>
        </Panel>
        <PanelResizeHandle
          style={{ width: 1, background: "var(--border)", cursor: "col-resize" }}
        />
        <Panel defaultSize={24} minSize={15} maxSize={40}>
          <DetailPanel selection={selection} onNavigate={selectFile} />
        </Panel>
      </PanelGroup>

      <StatusBar stats={data.stats} graph={data.graph} />
    </div>
  );
}

function renderView(
  view: ViewName,
  data: NonNullable<ReturnType<typeof useReportData>["data"]>,
  onModuleSelect: (id: string | null, node?: ModuleNode) => void,
  onFileSelect: (fp: string) => void,
) {
  switch (view) {
    case "architecture":
      return (
        <ArchitectureMap
          graph={data.graph}
          summaries={data.summaries}
          onNodeSelect={onModuleSelect}
        />
      );
    case "dependencies":
      return (
        <DependencyGraph
          graph={data.graph}
          summaries={data.summaries}
          onNodeSelect={onFileSelect}
        />
      );
    case "models":
      return <DataModels summaries={data.summaries} onNodeSelect={onFileSelect} />;
    case "flows":
      return (
        <CallFlow
          criticalPaths={data.architecture?.critical_paths || []}
          onNodeSelect={onFileSelect}
        />
      );
  }
}
