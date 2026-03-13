import React from "react";
import { createRoot } from "react-dom/client";
import "./styles/main.css";
import { App } from "./App";
import { ReportDataContext, useLoadReportData } from "./hooks/useReportData";

function Root() {
  const reportData = useLoadReportData();
  return (
    <ReportDataContext.Provider value={reportData}>
      <App />
    </ReportDataContext.Provider>
  );
}

const container = document.getElementById("app");
if (container) {
  createRoot(container).render(
    <React.StrictMode>
      <Root />
    </React.StrictMode>,
  );
}
