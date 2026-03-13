import React, { useState, useRef, useMemo, useCallback } from "react";
import Fuse from "fuse.js";
import type { ModuleSummary, SearchEntry } from "../types";
import { getLayerColor } from "../utils/layerColors";

interface SearchBarProps {
  summaries: ModuleSummary[];
  onSelect: (filePath: string) => void;
}

export function SearchBar({ summaries, onSelect }: SearchBarProps) {
  const [query, setQuery] = useState("");
  const [activeIndex, setActiveIndex] = useState(-1);
  const [showResults, setShowResults] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const hideTimer = useRef<ReturnType<typeof setTimeout>>();

  const { fuse, entries } = useMemo(() => {
    const entries: SearchEntry[] = summaries.map((s) => ({
      id: s.file_path,
      label: s.file_path.split("/").pop() || s.file_path,
      file_path: s.file_path,
      kind: s.layer,
      keywords: [
        s.summary,
        ...(s.responsibilities || []),
        ...(s.exports || []).map((e) => e.name),
        ...(s.patterns || []),
      ].join(" "),
    }));

    const fuse = new Fuse(entries, {
      keys: [
        { name: "label", weight: 2 },
        { name: "file_path", weight: 1.5 },
        { name: "keywords", weight: 1 },
      ],
      threshold: 0.4,
      includeScore: true,
    });

    return { fuse, entries };
  }, [summaries]);

  const results = useMemo(() => {
    if (!query.trim()) return [];
    return fuse.search(query, { limit: 10 }).map((r) => r.item);
  }, [fuse, query]);

  const handleSelect = useCallback(
    (filePath: string) => {
      onSelect(filePath);
      setQuery("");
      setShowResults(false);
    },
    [onSelect],
  );

  const handleKeydown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setActiveIndex((i) => Math.min(i + 1, results.length - 1));
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        setActiveIndex((i) => Math.max(i - 1, 0));
      } else if (e.key === "Enter" && activeIndex >= 0 && results[activeIndex]) {
        e.preventDefault();
        handleSelect(results[activeIndex].file_path);
      } else if (e.key === "Escape") {
        setQuery("");
        setShowResults(false);
        inputRef.current?.blur();
      }
    },
    [activeIndex, results, handleSelect],
  );

  return (
    <div className="search-bar">
      <span className="search-bar__icon">{"\u2315"}</span>
      <input
        ref={inputRef}
        className="search-bar__input"
        type="text"
        placeholder="Search files..."
        value={query}
        onChange={(e) => {
          setQuery(e.target.value);
          setActiveIndex(-1);
          setShowResults(true);
        }}
        onFocus={() => {
          clearTimeout(hideTimer.current);
          if (query.trim()) setShowResults(true);
        }}
        onBlur={() => {
          hideTimer.current = setTimeout(() => setShowResults(false), 150);
        }}
        onKeyDown={handleKeydown}
      />
      {showResults && results.length > 0 && (
        <div className="search-bar__results">
          {results.map((entry, i) => (
            <div
              key={entry.id}
              className={`search-bar__result ${i === activeIndex ? "search-bar__result--active" : ""}`}
              onMouseDown={() => handleSelect(entry.file_path)}
            >
              <span
                className="nav-tree__dot"
                style={{ backgroundColor: getLayerColor(entry.kind) }}
              />
              <span>{entry.label}</span>
              <span className="search-bar__result-path">{entry.file_path}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
