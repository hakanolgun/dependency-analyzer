import React from "react";
import {
  FileJson,
  Loader2,
  CheckCircle2,
  XCircle,
  ArrowUp,
  ArrowDown,
  RefreshCw,
  ExternalLink,
  Download,
} from "lucide-react";
import type { PackageResult } from "../lib/analyzer-js";

interface NpmResultsTableProps {
  results: PackageResult[];
  isAnalyzing: boolean;
  progress: number;
  hasReactNative: boolean;
  onReset: () => void;
}

type SortKey = "weeklyDownloads" | "lastUpdateDate" | "isMaintained" | "newArchitecture" | null;

export function NpmResultsTable({
  results,
  isAnalyzing,
  progress,
  hasReactNative,
  onReset,
}: NpmResultsTableProps) {
  const [sortKey, setSortKey] = React.useState<SortKey>(null);
  const [sortDirection, setSortDirection] = React.useState<"asc" | "desc">("desc");

  const renderBadge = (status?: string | boolean, type: string = "info", text?: string) => {
    if (status === undefined || status === null) return null;
    let badgeClass = `badge ${type}`;
    let display = text || String(status);

    if (typeof status === "boolean") {
      badgeClass = `badge ${status ? "success" : "danger"}`;
      display = status ? "Yes" : "No";
    }

    return <span className={badgeClass}>{display}</span>;
  };

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

  const formatWeeklyDownloads = (downloads?: number) => {
    if (downloads === undefined || downloads === null) return "-";
    return `${Math.floor(downloads / 1000)} K`;
  };

  const sortedResults = React.useMemo(() => {
    if (!sortKey) return results;

    return [...results].sort((a, b) => {
      let valA: string | number | boolean | undefined = a[sortKey] as unknown as
        | string
        | number
        | boolean
        | undefined;
      let valB: string | number | boolean | undefined = b[sortKey] as unknown as
        | string
        | number
        | boolean
        | undefined;

      if (sortKey === "isMaintained") {
        const weight = { yes: 3, unlikely: 2, no: 1 };
        valA = weight[a.isMaintained as keyof typeof weight] || 0;
        valB = weight[b.isMaintained as keyof typeof weight] || 0;
      } else if (sortKey === "lastUpdateDate") {
        valA = a.lastUpdateDate ? new Date(a.lastUpdateDate).getTime() : 0;
        valB = b.lastUpdateDate ? new Date(b.lastUpdateDate).getTime() : 0;
      } else if (sortKey === "newArchitecture") {
        valA = a.newArchitecture ? 1 : 0;
        valB = b.newArchitecture ? 1 : 0;
      }

      if (valA === undefined) valA = 0;
      if (valB === undefined) valB = 0;

      if (valA < valB) return sortDirection === "asc" ? -1 : 1;
      if (valA > valB) return sortDirection === "asc" ? 1 : -1;
      return 0;
    });
  }, [results, sortKey, sortDirection]);

  const handleDownloadResults = () => {
    const timestamp = new Date().toISOString().replace(/[:.]/g, "-");
    const payload = {
      generatedAt: new Date().toISOString(),
      ecosystem: "npm",
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
    link.download = `npm-analysis-results-${timestamp}.json`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
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
            Analyzed {results.length} packages{" "}
            {hasReactNative && (
              <span className="badge info" style={{ marginLeft: "0.5rem" }}>
                React Native Project Detected
              </span>
            )}
          </p>
        </div>

        <div style={{ display: "flex", gap: "0.5rem" }}>
          <button
            className="btn"
            onClick={handleDownloadResults}
            style={{ background: "rgba(255,255,255,0.1)" }}>
            <Download size={18} /> Export JSON
          </button>
          <button
            className="btn"
            onClick={onReset}
            style={{ background: "rgba(255,255,255,0.1)" }}>
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
                style={{ cursor: "pointer", userSelect: "none" }}
                title="Click to reset to original package.json order">
                <div style={{ display: "flex", alignItems: "center", gap: "4px" }}>
                  Package Name
                </div>
              </th>
              <th>Your Version</th>
              <th>Latest Version</th>
              <th
                onClick={() => handleSort("weeklyDownloads")}
                style={{ cursor: "pointer", userSelect: "none" }}>
                <div style={{ display: "flex", alignItems: "center", gap: "4px" }}>
                  Weekly Downloads {renderSortIcon("weeklyDownloads")}
                </div>
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
              {hasReactNative && (
                <th
                  onClick={() => handleSort("newArchitecture")}
                  style={{ cursor: "pointer", userSelect: "none" }}>
                  <div style={{ display: "flex", alignItems: "center", gap: "4px" }}>
                    New Arch Support {renderSortIcon("newArchitecture")}
                  </div>
                </th>
              )}
            </tr>
          </thead>
          <tbody>
            {sortedResults.map((pkg, idx) => (
              <tr key={`${pkg.name}-${idx}`}>
                <td>
                  <div className="pkg-name">
                    <FileJson size={16} color="var(--primary)" />
                    {pkg.repoUrl ? (
                      <a
                        href={pkg.repoUrl}
                        target="_blank"
                        rel="noreferrer"
                        style={{ color: "inherit", textDecoration: "none", wordBreak: "break-all" }}
                        className="repo-link"
                        title={pkg.repoUrl}>
                        {pkg.name}
                        <ExternalLink size={12} style={{ marginLeft: "4px", opacity: 0.6 }} />
                      </a>
                    ) : (
                      <span style={{ wordBreak: "break-all" }}>{pkg.name}</span>
                    )}
                  </div>
                  {pkg.status === "error" && (
                    <span
                      style={{
                        color: "var(--danger)",
                        fontSize: "0.8rem",
                        marginTop: "4px",
                        display: "block",
                      }}>
                      {pkg.error}
                    </span>
                  )}
                </td>
                <td>
                  <span className="badge gray">
                    {pkg.currentVersion.replace(/[\^~]/, "")}
                  </span>
                </td>
                <td>
                  {pkg.status === "loading" ? (
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
                  ) : pkg.latestVersion ? (
                    renderBadge(pkg.latestVersion, "info")
                  ) : (
                    "-"
                  )}
                </td>
                <td>
                  {pkg.status === "loading"
                    ? "-"
                    : formatWeeklyDownloads(pkg.weeklyDownloads)}
                </td>
                <td>
                  {pkg.status === "loading" ? (
                    "-"
                  ) : pkg.timeSinceLastUpdate ? (
                    <div style={{ display: "flex", flexDirection: "column", gap: "2px" }}>
                      <span>{pkg.timeSinceLastUpdate}</span>
                      <span style={{ fontSize: "0.75rem", color: "var(--text-muted)" }}>
                        {new Date(pkg.lastUpdateDate!).toLocaleDateString()}
                      </span>
                    </div>
                  ) : (
                    "-"
                  )}
                </td>
                <td>
                  {pkg.status === "loading" ? "-" : renderMaintainedStatus(pkg.isMaintained)}
                </td>
                <td>
                  {pkg.status === "loading"
                    ? "-"
                    : <span> See notes below </span> 
                    }
                </td>
                {hasReactNative && (
                  <td>
                    {pkg.newArchitecture !== undefined ? (
                      pkg.newArchitecture ? (
                        <CheckCircle2 size={20} color="var(--success)" />
                      ) : (
                        <XCircle size={20} color="var(--danger)" />
                      )
                    ) : (
                      "-"
                    )}
                  </td>
                )}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
}
