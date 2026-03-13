import React, { useState, useMemo, useCallback } from "react";
import type { ModuleSummary, LayerType } from "../types";
import { getLayerColor, LAYER_HINTS } from "../utils/layerColors";

interface NavTreeProps {
  summaries: ModuleSummary[];
  selectedPath: string | null;
  onSelect: (filePath: string) => void;
}

interface TreeGroup {
  name: string;
  moduleId: string;
  layer: LayerType;
  items: ModuleSummary[];
}

function getModuleDir(filePath: string): string {
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

function getModuleLabel(moduleId: string): string {
  const parts = moduleId.split("/");
  if (parts.length >= 3 && (parts[1] === "internal" || parts[1] === "src")) {
    return parts.slice(2).join("/");
  }
  if (parts.length >= 2) return parts[parts.length - 1];
  return moduleId;
}

export function NavTree({ summaries, selectedPath, onSelect }: NavTreeProps) {
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});

  const groups = useMemo(() => {
    const moduleGroups = new Map<string, ModuleSummary[]>();
    for (const s of summaries) {
      const moduleId = getModuleDir(s.file_path);
      if (!moduleGroups.has(moduleId)) moduleGroups.set(moduleId, []);
      moduleGroups.get(moduleId)!.push(s);
    }

    return Array.from(moduleGroups.entries())
      .map(([moduleId, items]): TreeGroup => {
        const dirName = moduleId.split("/").pop() || moduleId;
        const layer = LAYER_HINTS[dirName.toLowerCase()] || "unknown";
        return {
          name: getModuleLabel(moduleId),
          moduleId,
          layer,
          items: items.sort((a, b) => a.file_path.localeCompare(b.file_path)),
        };
      })
      .sort((a, b) => a.name.localeCompare(b.name));
  }, [summaries]);

  const toggleGroup = useCallback((moduleId: string) => {
    setCollapsed((prev) => ({ ...prev, [moduleId]: !prev[moduleId] }));
  }, []);

  // Start all collapsed by default
  const isCollapsed = (moduleId: string) => collapsed[moduleId] !== false;

  return (
    <div className="nav-tree">
      {groups.map((group) => (
        <div key={group.moduleId} className="nav-tree__group">
          <div className="nav-tree__group-header" onClick={() => toggleGroup(group.moduleId)}>
            <span
              className={`nav-tree__chevron ${isCollapsed(group.moduleId) ? "nav-tree__chevron--collapsed" : ""}`}
            >
              {"\u25BE"}
            </span>
            <span
              className="nav-tree__dot"
              style={{ backgroundColor: getLayerColor(group.layer) }}
            />
            <span>
              {group.name} ({group.items.length})
            </span>
          </div>
          {!isCollapsed(group.moduleId) &&
            group.items.map((item) => (
              <div
                key={item.file_path}
                className={`nav-tree__item ${item.file_path === selectedPath ? "nav-tree__item--selected" : ""}`}
                onClick={() => onSelect(item.file_path)}
              >
                <span
                  className="nav-tree__dot"
                  style={{ backgroundColor: getLayerColor(group.layer) }}
                />
                <span title={item.file_path}>
                  {item.file_path.split("/").pop() || item.file_path}
                </span>
              </div>
            ))}
        </div>
      ))}
    </div>
  );
}
