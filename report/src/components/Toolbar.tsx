import React from "react";

export type ViewName = "architecture" | "dependencies" | "models" | "flows";

const TABS: { id: ViewName; label: string }[] = [
  { id: "architecture", label: "Architecture" },
  { id: "dependencies", label: "Dependencies" },
  { id: "models", label: "Data Models" },
  { id: "flows", label: "Call Flow" },
];

interface ToolbarProps {
  repoName: string;
  activeView: ViewName;
  onViewChange: (view: ViewName) => void;
  children?: React.ReactNode;
}

export function Toolbar({ repoName, activeView, onViewChange, children }: ToolbarProps) {
  return (
    <div className="toolbar">
      <div className="toolbar__logo">repomap</div>
      <span className="toolbar__repo-name">{repoName}</span>
      <div className="toolbar__tabs">
        {TABS.map((tab) => (
          <button
            key={tab.id}
            className={`toolbar__tab ${activeView === tab.id ? "toolbar__tab--active" : ""}`}
            onClick={() => onViewChange(tab.id)}
          >
            {tab.label}
          </button>
        ))}
      </div>
      <div className="toolbar__spacer" />
      {children}
    </div>
  );
}
