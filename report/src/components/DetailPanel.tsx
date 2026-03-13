import React, { useEffect, useState } from "react";
import type { ModuleSummary } from "../types";
import type { ModuleNode } from "../utils/moduleGrouper";
import { getLayerColor } from "../utils/layerColors";

interface DetailPanelProps {
  selection:
    | { type: "file"; summary: ModuleSummary }
    | { type: "module"; moduleNode: ModuleNode; summaryMap: Map<string, ModuleSummary> }
    | null;
  onNavigate: (filePath: string) => void;
}

type Highlighter = {
  codeToHtml: (code: string, opts: { lang: string; theme: string }) => string;
} | null;

export function DetailPanel({ selection, onNavigate }: DetailPanelProps) {
  const [highlighter, setHighlighter] = useState<Highlighter>(null);

  useEffect(() => {
    import("shiki")
      .then((shiki) =>
        shiki.createHighlighter({
          themes: ["vitesse-dark"],
          langs: ["typescript", "javascript", "python", "go", "rust", "java", "ruby"],
        }),
      )
      .then(setHighlighter)
      .catch(() => setHighlighter(null));
  }, []);

  if (!selection) {
    return (
      <div className="detail-panel">
        <div className="detail-panel__empty">
          Select a module from the graph or nav tree to view details
        </div>
      </div>
    );
  }

  if (selection.type === "module") {
    return (
      <div className="detail-panel">
        <ModuleDetail
          moduleNode={selection.moduleNode}
          summaryMap={selection.summaryMap}
          onNavigate={onNavigate}
        />
      </div>
    );
  }

  return (
    <div className="detail-panel">
      <FileDetail summary={selection.summary} onNavigate={onNavigate} highlighter={highlighter} />
    </div>
  );
}

function FileDetail({
  summary,
  onNavigate,
  highlighter,
}: {
  summary: ModuleSummary;
  onNavigate: (fp: string) => void;
  highlighter: Highlighter;
}) {
  const langMap: Record<string, string> = {
    typescript: "typescript",
    javascript: "javascript",
    python: "python",
    go: "go",
    rust: "rust",
    java: "java",
    ruby: "ruby",
  };

  return (
    <>
      <Header layer={summary.layer} title={summary.file_path} />
      <Section title="Summary">
        <div className="detail-panel__summary">{summary.summary}</div>
      </Section>
      {summary.responsibilities.length > 0 && (
        <Section title="Responsibilities">
          <List items={summary.responsibilities} />
        </Section>
      )}
      {summary.patterns.length > 0 && (
        <Section title="Patterns">
          <List items={summary.patterns} />
        </Section>
      )}
      {summary.exports.length > 0 && (
        <Section title="Exports">
          <List items={summary.exports.map((e) => `${e.name} (${e.kind}, line ${e.line})`)} />
        </Section>
      )}
      {summary.dependencies_on.length > 0 && (
        <Section title="Depends On">
          {summary.dependencies_on.map((dep) => (
            <a
              key={dep}
              className="detail-panel__dep"
              href="#"
              onClick={(e) => {
                e.preventDefault();
                onNavigate(dep);
              }}
            >
              {dep}
            </a>
          ))}
        </Section>
      )}
      {summary.depended_on_by_hint && summary.depended_on_by_hint.length > 0 && (
        <Section title="Depended On By">
          {summary.depended_on_by_hint.map((dep) => (
            <a
              key={dep}
              className="detail-panel__dep"
              href="#"
              onClick={(e) => {
                e.preventDefault();
                onNavigate(dep);
              }}
            >
              {dep}
            </a>
          ))}
        </Section>
      )}
      {summary.source_code && (
        <Section title="Source Code">
          <CodeBlock
            code={summary.source_code}
            language={langMap[summary.language.toLowerCase()] || "typescript"}
            highlighter={highlighter}
          />
        </Section>
      )}
    </>
  );
}

function ModuleDetail({
  moduleNode,
  summaryMap,
  onNavigate,
}: {
  moduleNode: ModuleNode;
  summaryMap: Map<string, ModuleSummary>;
  onNavigate: (fp: string) => void;
}) {
  const allExports: string[] = [];
  for (const fp of moduleNode.files) {
    const s = summaryMap.get(fp);
    if (s?.exports) {
      for (const exp of s.exports) {
        allExports.push(`${exp.name} (${exp.kind})`);
      }
    }
  }

  return (
    <>
      <Header layer={moduleNode.layer} title={moduleNode.label} />
      <Section title={`${moduleNode.fileCount} Files`}>
        {moduleNode.files.map((fp) => (
          <a
            key={fp}
            className="detail-panel__dep"
            href="#"
            title={fp}
            onClick={(e) => {
              e.preventDefault();
              onNavigate(fp);
            }}
          >
            {fp.split("/").pop() || fp}
          </a>
        ))}
      </Section>
      {moduleNode.summary && (
        <Section title="Summary">
          <div className="detail-panel__summary">{moduleNode.summary}</div>
        </Section>
      )}
      {allExports.length > 0 && (
        <Section title="Exports">
          <List items={allExports.slice(0, 15)} />
          {allExports.length > 15 && (
            <div style={{ color: "#64748b", fontSize: 12, marginTop: 4 }}>
              ... and {allExports.length - 15} more
            </div>
          )}
        </Section>
      )}
    </>
  );
}

function Header({ layer, title }: { layer: string; title: string }) {
  return (
    <div className="detail-panel__header">
      <span
        className="detail-panel__badge"
        style={{ backgroundColor: getLayerColor(layer), color: "#fff" }}
      >
        {layer}
      </span>
      <div className="detail-panel__title">{title}</div>
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="detail-panel__section">
      <div className="detail-panel__section-title">{title}</div>
      {children}
    </div>
  );
}

function List({ items }: { items: string[] }) {
  return (
    <ul className="detail-panel__list">
      {items.map((item, i) => (
        <li key={i}>{item}</li>
      ))}
    </ul>
  );
}

function CodeBlock({
  code,
  language,
  highlighter,
}: {
  code: string;
  language: string;
  highlighter: Highlighter;
}) {
  if (highlighter) {
    try {
      const html = highlighter.codeToHtml(code, { lang: language, theme: "vitesse-dark" });
      return <div className="detail-panel__code" dangerouslySetInnerHTML={{ __html: html }} />;
    } catch {
      // fall through to plain text
    }
  }
  return (
    <div className="detail-panel__code">
      <pre>{code}</pre>
    </div>
  );
}
