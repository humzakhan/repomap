import { useState, useEffect, createContext, useContext } from "react";
import type { RepoMapData, ModuleSummary } from "../types";

interface ReportDataState {
  data: RepoMapData | null;
  summaryMap: Map<string, ModuleSummary>;
  loading: boolean;
  error: string | null;
}

export const ReportDataContext = createContext<ReportDataState>({
  data: null,
  summaryMap: new Map(),
  loading: true,
  error: null,
});

export function useReportData() {
  return useContext(ReportDataContext);
}

export function useLoadReportData(): ReportDataState {
  const [state, setState] = useState<ReportDataState>({
    data: null,
    summaryMap: new Map(),
    loading: true,
    error: null,
  });

  useEffect(() => {
    loadData()
      .then((data) => {
        const summaryMap = new Map<string, ModuleSummary>();
        for (const s of data.summaries) {
          summaryMap.set(s.file_path, s);
        }
        setState({ data, summaryMap, loading: false, error: null });
      })
      .catch((err) => {
        setState({ data: null, summaryMap: new Map(), loading: false, error: String(err) });
      });
  }, []);

  return state;
}

async function loadData(): Promise<RepoMapData> {
  // Try API endpoint first (local server mode)
  try {
    const resp = await fetch("/api/report");
    if (resp.ok) {
      return await resp.json();
    }
  } catch {
    // Not running in server mode, fall through
  }

  // Try embedded JSON (static HTML mode)
  const scriptTag = document.getElementById("repomap-data");
  if (scriptTag && scriptTag.textContent && scriptTag.textContent.trim().length > 2) {
    return JSON.parse(scriptTag.textContent);
  }

  // Dev mode: load fixture
  const fixture = await import("../fixtures/sample.json");
  return fixture.default as unknown as RepoMapData;
}
