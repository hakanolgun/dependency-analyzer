import React, { useState, useCallback } from "react";
import {
  Upload,
  ArrowRight,
  AlertCircle,
  ChevronDown,
} from "lucide-react";
import { parsePackageJson, fetchPackageData, type PackageResult } from "./lib/analyzer-js";
import { parseGoMod, fetchGoModuleData, type GoModuleResult } from "./lib/analyzer-go";
import { NpmResultsTable } from "./components/NpmResultsTable";
import { GoResultsTable } from "./components/GoResultsTable";
import { MaintenanceLegend } from "./components/MaintenanceLegend";
import { Footer } from "./Footer";

type Ecosystem = "npm" | "go";

function App() {
  const [ecosystem, setEcosystem] = useState<Ecosystem>("npm");
  const [inputVal, setInputVal] = useState("");
  const [isAnalyzing, setIsAnalyzing] = useState(false);
  const [npmResults, setNpmResults] = useState<PackageResult[]>([]);
  const [goResults, setGoResults] = useState<GoModuleResult[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [dragActive, setDragActive] = useState(false);
  const [progress, setProgress] = useState(0);
  const [hasReactNative, setHasReactNative] = useState(false);

  const hasResults = ecosystem === "npm" ? npmResults.length > 0 : goResults.length > 0;

  const handleAnalyzeNpm = async (content: string) => {
    const parsed = parsePackageJson(content);
    if (!parsed) {
      setError("Invalid JSON format. Please pass a valid package.json content.");
      return;
    }

    const allDeps = { ...parsed.dependencies, ...parsed.devDependencies };
    const depEntries = Object.entries(allDeps);

    if (depEntries.length === 0) {
      setError("No dependencies found in the provided package.json.");
      return;
    }

    const isRN = Object.keys(allDeps).includes("react-native");
    setHasReactNative(isRN);

    setIsAnalyzing(true);
    setNpmResults([]);
    setProgress(0);

    const newResults: PackageResult[] = [];
    const batchSize = 5;

    for (let i = 0; i < depEntries.length; i += batchSize) {
      const batch = depEntries.slice(i, i + batchSize);
      const promises = batch.map(([name, version]) => fetchPackageData(name, version, isRN));
      const batchResults = await Promise.all(promises);
      newResults.push(...batchResults);
      setProgress(Math.round(((i + batch.length) / depEntries.length) * 100));
      setNpmResults([...newResults]);
    }

    setIsAnalyzing(false);
  };

  const handleAnalyzeGo = async (content: string) => {
    const parsed = parseGoMod(content);
    if (!parsed) {
      setError("Invalid go.mod format. Please pass a valid go.mod content.");
      return;
    }

    if (parsed.dependencies.length === 0) {
      setError("No direct dependencies found in the provided go.mod.");
      return;
    }

    setIsAnalyzing(true);
    setGoResults([]);
    setProgress(0);

    const newResults: GoModuleResult[] = [];
    const batchSize = 5;

    for (let i = 0; i < parsed.dependencies.length; i += batchSize) {
      const batch = parsed.dependencies.slice(i, i + batchSize);
      const promises = batch.map((dep) => fetchGoModuleData(dep.path, dep.version));
      const batchResults = await Promise.all(promises);
      newResults.push(...batchResults);
      setProgress(
        Math.round(((i + batch.length) / parsed.dependencies.length) * 100),
      );
      setGoResults([...newResults]);
    }

    setIsAnalyzing(false);
  };

  const handleAnalyze = useCallback(async (content: string) => {
    setError(null);
    if (ecosystem === "npm") {
      await handleAnalyzeNpm(content);
    } else {
      await handleAnalyzeGo(content);
    }
  }, [ecosystem]);

  const handleReset = () => {
    setNpmResults([]);
    setGoResults([]);
    setInputVal("");
    setError(null);
    setHasReactNative(false);
    setProgress(0);
  };

  const onDrag = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === "dragenter" || e.type === "dragover") {
      setDragActive(true);
    } else if (e.type === "dragleave") {
      setDragActive(false);
    }
  }, []);

  const onDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      setDragActive(false);
      if (e.dataTransfer.files && e.dataTransfer.files[0]) {
        const file = e.dataTransfer.files[0];
        const reader = new FileReader();
        reader.onload = (ev) => {
          const text = ev.target?.result as string;
          setInputVal(text);
          if (text) handleAnalyze(text);
        };
        reader.readAsText(file);
      }
    },
    [handleAnalyze],
  );

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      const file = e.target.files[0];
      const reader = new FileReader();
      reader.onload = (ev) => {
        const text = ev.target?.result as string;
        setInputVal(text);
        if (text) handleAnalyze(text);
      };
      reader.readAsText(file);
    }
  };

  const fileAccept = ecosystem === "npm" ? ".json" : ".mod";
  const fileName = ecosystem === "npm" ? "package.json" : "go.mod";
  const placeholder =
    ecosystem === "npm"
      ? '{"dependencies": {"react": "^18.2.0"}}'
      : 'module github.com/user/project\n\ngo 1.21\n\nrequire (\n    github.com/gin-gonic/gin v1.9.1\n)';

  return (
    <div className="app-container">
      <div className="glass-panel">
        {!hasResults && !isAnalyzing ? (
          <>
            <h1 className="title">Hakan's Dependency Analyzer</h1>
            <h2
              className="subtitle"
              style={{
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                gap: "0.5rem",
                marginTop: "-0.5rem",
              }}>
              <a
                href="https://github.com/hakanolgun/dependency-analyzer"
                target="_blank"
                rel="noreferrer"
                className="repo-link"
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: "0.5rem",
                  color: "inherit",
                  textDecoration: "none",
                  transition: "color 0.2s ease",
                }}>
                <svg
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round">
                  <path d="M9 19c-5 1.5-5-2.5-7-3m14 6v-3.87a3.37 3.37 0 0 0-.94-2.61c3.14-.35 6.44-1.54 6.44-7A5.44 5.44 0 0 0 20 4.77 5.07 5.07 0 0 0 19.91 1S18.73.65 16 2.48a13.38 13.38 0 0 0-7 0C6.27.65 5.09 1 5.09 1A5.07 5.07 0 0 0 5 4.77a5.44 5.44 0 0 0-1.5 3.78c0 5.42 3.3 6.61 6.44 7A3.37 3.37 0 0 0 9 18.13V22"></path>
                </svg>
                Free and Open Source
              </a>
            </h2>
            <p className="subtitle">
              Upload or paste your dependency file to get rich insights
            </p>

            {/* Ecosystem Selector */}
            <div className="ecosystem-selector">
              <label className="ecosystem-label">Select Ecosystem</label>
              <div className="custom-select-wrapper">
                <select
                  className="ecosystem-dropdown"
                  value={ecosystem}
                  onChange={(e) => {
                    setEcosystem(e.target.value as Ecosystem);
                    setInputVal("");
                    setError(null);
                  }}>
                  <option value="npm">JavaScript (package.json)</option>
                  <option value="go">Go (go.mod)</option>
                </select>
                <ChevronDown size={18} className="select-icon" />
              </div>
            </div>

            <label
              className={`upload-area ${dragActive ? "drag-active" : ""}`}
              onDragEnter={onDrag}
              onDragLeave={onDrag}
              onDragOver={onDrag}
              onDrop={onDrop}>
              <Upload className="upload-icon" />
              <div className="upload-text">Drag & Drop <strong>{fileName}</strong> here</div>
              <div className="upload-hint">or click to browse files</div>
              <input
                type="file"
                accept={fileAccept}
                style={{ display: "none" }}
                onChange={handleFileChange}
              />
            </label>

            <div className="text-area-container">
              <div
                style={{
                  textAlign: "center",
                  color: "var(--text-muted)",
                  margin: "0.5rem 0",
                }}>
                — OR PASTE CONTENT —
              </div>
              <textarea
                className="json-input"
                placeholder={placeholder}
                value={inputVal}
                onChange={(e) => setInputVal(e.target.value)}
              />
              {error && (
                <div
                  style={{
                    color: "var(--danger)",
                    display: "flex",
                    alignItems: "center",
                    gap: "0.5rem",
                    marginTop: "0.5rem",
                  }}>
                  <AlertCircle size={18} /> {error}
                </div>
              )}
              <button
                className="btn"
                style={{ alignSelf: "center", marginTop: "1rem" }}
                onClick={() => handleAnalyze(inputVal)}
                disabled={!inputVal.trim()}>
                Analyze Dependencies <ArrowRight size={18} />
              </button>
            </div>
          </>
        ) : isAnalyzing && !hasResults ? (
          <div className="loader-container">
            <div className="spinner"></div>
            <h2 style={{ fontSize: "1.5rem" }}>
              Analyzing {ecosystem === "npm" ? "packages" : "modules"}...
            </h2>
            <div className="progress-bar">
              <div className="progress-fill" style={{ width: `${progress}%` }}></div>
            </div>
            <p style={{ color: "var(--text-muted)" }}>{progress}% Complete</p>
          </div>
        ) : ecosystem === "npm" ? (
          <NpmResultsTable
            results={npmResults}
            isAnalyzing={isAnalyzing}
            progress={progress}
            hasReactNative={hasReactNative}
            onReset={handleReset}
          />
        ) : (
          <GoResultsTable
            results={goResults}
            isAnalyzing={isAnalyzing}
            progress={progress}
            onReset={handleReset}
          />
        )}
      </div>

      <MaintenanceLegend />

      <Footer />
    </div>
  );
}

export default App;
