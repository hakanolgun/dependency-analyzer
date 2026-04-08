import React from "react";
import {
  Loader2,
  ArrowUp,
  ArrowDown,
  RefreshCw,
  ExternalLink,
  Package,
  Download,
} from "lucide-react";
import type { GoModuleResult } from "../lib/analyzer-go";
import { normalizeRepoUrl } from "../lib/utils";

interface GoResultsTableProps {
  results: GoModuleResult[];
  isAnalyzing: boolean;
  progress: number;
  onReset: () => void;
}

type SortKey = "lastUpdateDate" | "isMaintained" | null;

export function GoResultsTable({ results, isAnalyzing, progress, onReset }: GoResultsTableProps) {
  const [sortKey, setSortKey] = React.useState<SortKey>(null);
  const [sortDirection, setSortDirection] = React.useState<"asc" | "desc">("desc");

  const renderMaintainedStatus = (status?: "yes" | "unlikely" | "no") => {
    if (!status) return "-";
    let type = "success";
    if (status === "no") type = "danger";
    if (status === "unlikely") type = "warning";

    const display = status === "yes" ? "Yes" : status === "unlikely" ? "Unlikely" : "No";

    return <span className={`badge ${type}`}>{display}</span>;
  };

  const handleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDirection((prev) => (prev === "asc" ? "desc" : "asc"));
    } else {
      setSortKey(key);
      setSortDirection("desc");
    }
  };

  const renderSortIcon = (key: SortKey) => {
    if (sortKey !== key) return null;
    return sortDirection === "asc" ? <ArrowUp size={14} /> : <ArrowDown size={14} />;
  };

  const sortedResults = React.useMemo(() => {
    if (!sortKey) return results;

    return [...results].sort((a, b) => {
      let valA: number = 0;
      let valB: number = 0;

      if (sortKey === "isMaintained") {
        const weight = { yes: 3, unlikely: 2, no: 1 };
        valA = weight[a.isMaintained as keyof typeof weight] || 0;
        valB = weight[b.isMaintained as keyof typeof weight] || 0;
      } else if (sortKey === "lastUpdateDate") {
        valA = a.lastUpdateDate ? new Date(a.lastUpdateDate).getTime() : 0;
        valB = b.lastUpdateDate ? new Date(b.lastUpdateDate).getTime() : 0;
      }

      if (valA < valB) return sortDirection === "asc" ? -1 : 1;
      if (valA > valB) return sortDirection === "asc" ? 1 : -1;
      return 0;
    });
  }, [results, sortKey, sortDirection]);

  const handleDownloadResults = () => {
    const timestamp = new Date().toISOString().replace(/[:.]/g, "-");
    const payload = {
      generatedAt: new Date().toISOString(),
      ecosystem: "go",
      sort: {
        key: sortKey,
        direction: sortDirection,
      },
      results: sortedResults,
    };

    const blob = new Blob([JSON.stringify(payload, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `go-analysis-results-${timestamp}.json`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  };

  /**
   * Check if current version differs from latest version.
   * Strips leading 'v' and compares.
   */
  const isOutdated = (current: string, latest?: string) => {
    if (!latest) return false;
    const clean = (v: string) => v.replace(/^v/, "");
    return clean(current) !== clean(latest);
  };

  return (
    <>
      <div className="header-actions">
        <div>
          <h1
            className="title"
            style={{ textAlign: "left", fontSize: "2rem", marginBottom: "0.2rem" }}>
            Analysis Complete
          </h1>
          <p className="subtitle" style={{ textAlign: "left", margin: 0 }}>
            Analyzed {results.length} modules{" "}
            <span className="badge go" style={{ marginLeft: "0.5rem" }}>
              Go Module
            </span>
          </p>
        </div>

        <div style={{ display: "flex", gap: "0.5rem" }}>
          <button
            className="btn"
            onClick={handleDownloadResults}
            style={{ background: "rgba(255,255,255,0.1)" }}>
            <Download size={18} /> Export JSON
          </button>
          <button className="btn" onClick={onReset} style={{ background: "rgba(255,255,255,0.1)" }}>
            <RefreshCw size={18} /> Analyze Another
          </button>
        </div>
      </div>

      {isAnalyzing && (
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: "1rem",
            marginBottom: "1.5rem",
          }}>
          <Loader2
            size={20}
            className="spinner"
            style={{ width: 20, height: 20, borderWidth: 2 }}
          />
          <div className="progress-bar" style={{ flex: 1, maxWidth: "none", height: 4 }}>
            <div className="progress-fill" style={{ width: `${progress}%` }}></div>
          </div>
        </div>
      )}

      <div className="table-container">
        <table>
          <thead>
            <tr>
              <th
                onClick={() => setSortKey(null)}
                style={{ cursor: "pointer", userSelect: "none", minWidth: "70px" }}
                title="Click to reset to original go.mod order">
                <div style={{ display: "flex", alignItems: "center", gap: "4px" }}>Module Name</div>
              </th>
              <th
                onClick={() => handleSort("lastUpdateDate")}
                style={{ cursor: "pointer", userSelect: "none" }}>
                <div style={{ display: "flex", alignItems: "center", gap: "4px" }}>
                  Last Update {renderSortIcon("lastUpdateDate")}
                </div>
              </th>
              <th
                onClick={() => handleSort("isMaintained")}
                style={{ cursor: "pointer", userSelect: "none" }}>
                <div style={{ display: "flex", alignItems: "center", gap: "4px" }}>
                  Maintained {renderSortIcon("isMaintained")}
                </div>
              </th>
              <th>Replaceability</th>
              <th>Your Version</th>
              <th>Latest Version</th>
            </tr>
          </thead>
          <tbody>
            {sortedResults.map((mod, idx) => (
              <tr key={`${mod.name}-${idx}`}>
                <td style={{ minWidth: "70px" }}>
                  <div className="pkg-name">
                    <Package size={16} color="var(--go-color)" />
                    {mod.repoUrl ? (
                      <a
                        href={normalizeRepoUrl(mod.repoUrl)}
                        target="_blank"
                        rel="noreferrer"
                        style={{ color: "inherit", textDecoration: "none", wordBreak: "break-all" }}
                        className="repo-link"
                        title={mod.repoUrl}>
                        {mod.name}
                        <ExternalLink size={12} style={{ marginLeft: "4px", opacity: 0.6 }} />
                      </a>
                    ) : (
                      <span style={{ wordBreak: "break-all" }}>{mod.name}</span>
                    )}
                  </div>
                  {mod.status === "error" && (
                    <span
                      style={{
                        color: "var(--danger)",
                        fontSize: "0.8rem",
                        marginTop: "4px",
                        display: "block",
                      }}>
                      {mod.error}
                    </span>
                  )}
                </td>
                <td>
                  {mod.status === "loading" ? (
                    "-"
                  ) : mod.timeSinceLastUpdate ? (
                    <div style={{ display: "flex", flexDirection: "column", gap: "2px" }}>
                      <span>{mod.timeSinceLastUpdate}</span>
                      <span style={{ fontSize: "0.75rem", color: "var(--text-muted)" }}>
                        {new Date(mod.lastUpdateDate!).toLocaleDateString()}
                      </span>
                    </div>
                  ) : (
                    "-"
                  )}
                </td>
                <td>{mod.status === "loading" ? "-" : renderMaintainedStatus(mod.isMaintained)}</td>
                <td>
                  {mod.status === "loading" ? (
                    "-"
                  ) : (
                    <a href="#replaceability-section" className="replace-notes-link">
                      See notes below
                    </a>
                  )}
                </td>
                <td>
                  <span className="badge gray">{mod.currentVersion}</span>
                </td>
                <td>
                  {mod.status === "loading" ? (
                    <Loader2
                      size={16}
                      className="spinner"
                      style={{
                        width: 16,
                        height: 16,
                        borderWidth: 2,
                        borderColor: "var(--text-muted)",
                        borderLeftColor: "transparent",
                      }}
                    />
                  ) : mod.latestVersion ? (
                    <span
                      className={`badge ${isOutdated(mod.currentVersion, mod.latestVersion) ? "warning" : "info"}`}>
                      {mod.latestVersion}
                    </span>
                  ) : (
                    "-"
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
}
