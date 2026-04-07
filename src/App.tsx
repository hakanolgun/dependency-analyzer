import React, { useState, useCallback } from "react";
import { Upload, ArrowRight, AlertCircle } from "lucide-react";
import { parsePackageJson, fetchPackageData, type PackageResult } from "./lib/analyzer-js";
import { parseGoMod, fetchGoModuleData, type GoModuleResult } from "./lib/analyzer-go";
import { NpmResultsTable } from "./components/NpmResultsTable";
import { GoResultsTable } from "./components/GoResultsTable";
import { MaintenanceLegend } from "./components/MaintenanceLegend";
import { ReplaceabilityLegend } from "./components/ReplaceabilityLegend";
import { Footer } from "./Footer";
import { CopyCommand, npxCommand } from "./components/CopyCommand";
import { HomeTitle } from "./components/HomeTitle";

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

  const handleAnalyzeNpm = useCallback(async (content: string) => {
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
  }, []);

  const handleAnalyzeGo = useCallback(async (content: string) => {
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
      setProgress(Math.round(((i + batch.length) / parsed.dependencies.length) * 100));
      setGoResults([...newResults]);
    }

    setIsAnalyzing(false);
  }, []);

  const handleAnalyze = useCallback(
    async (content: string, explicitEcosystem?: Ecosystem) => {
      setError(null);
      let targetEcosystem: Ecosystem | null = explicitEcosystem || null;

      if (!targetEcosystem) {
        const trimmed = content.trim();
        // Detect NPM
        if (trimmed.startsWith("{")) {
          try {
            const parsed = JSON.parse(trimmed);
            if (parsed.dependencies || parsed.devDependencies) {
              targetEcosystem = "npm";
            }
          } catch {
            // ignore
          }
        }
        // Detect Go
        if (!targetEcosystem) {
          const hasModuleLine = /^\s*module\s+\S+/.test(trimmed);
          const hasRequireLine = /^\s*require\s*\(?/.test(trimmed);
          const hasGoDirective = /^\s*go\s+\d+\.\d+/.test(trimmed);
          if (hasModuleLine || hasRequireLine || hasGoDirective) {
            targetEcosystem = "go";
          }
        }
      }

      if (!targetEcosystem) {
        setError("Could not detect file type. Please upload a valid package.json or go.mod.");
        return;
      }

      setEcosystem(targetEcosystem);
      if (targetEcosystem === "npm") {
        await handleAnalyzeNpm(content);
      } else {
        await handleAnalyzeGo(content);
      }
    },
    [handleAnalyzeNpm, handleAnalyzeGo],
  );

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
      setError(null);

      if (e.dataTransfer.files && e.dataTransfer.files[0]) {
        const file = e.dataTransfer.files[0];
        const name = file.name.toLowerCase();

        if (name !== "package.json" && name !== "go.mod") {
          setError("Unsupported file. Please upload package.json or go.mod.");
          return;
        }

        const explicitEco = name === "package.json" ? "npm" : "go";

        const reader = new FileReader();
        reader.onload = (ev) => {
          const text = ev.target?.result as string;
          setInputVal(text);
          if (text) handleAnalyze(text, explicitEco);
        };
        reader.readAsText(file);
      }
    },
    [handleAnalyze],
  );

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setError(null);
    if (e.target.files && e.target.files[0]) {
      const file = e.target.files[0];
      const name = file.name.toLowerCase();

      if (name !== "package.json" && name !== "go.mod") {
        setError("Unsupported file. Please upload package.json or go.mod.");
        return;
      }

      const explicitEco = name === "package.json" ? "npm" : "go";

      const reader = new FileReader();
      reader.onload = (ev) => {
        const text = ev.target?.result as string;
        setInputVal(text);
        if (text) handleAnalyze(text, explicitEco);
      };
      reader.readAsText(file);
    }
  };

  const fileAccept = ".json,.mod";
  const placeholder =
    'Paste package.json or go.mod content here...\n\ne.g. {"dependencies": {"react": "^18.2.0"}}\n\nOR\n\nrequire github.com/gin-gonic/gin v1.9.1';

  return (
    <div className="app-container">
      <div className="glass-panel">
        {!hasResults && !isAnalyzing ? (
          <>
            <HomeTitle />

            <p className="or-divider">
              Run this in your project directory to get a complete report
            </p>
            <CopyCommand command={npxCommand} />
            <p style={{ paddingBlock: "2rem" }} className="or-divider">
              — or to get a basic report without analysis —
            </p>

            <label
              className={`upload-area ${dragActive ? "drag-active" : ""}`}
              onDragEnter={onDrag}
              onDragLeave={onDrag}
              onDragOver={onDrag}
              onDrop={onDrop}>
              <Upload className="upload-icon" />
              <div className="upload-text">
                Upload <strong>package.json</strong> or <strong>go.mod</strong> file here
              </div>

              <div className="upload-hint">or click to browse files</div>
              <input
                type="file"
                accept={fileAccept}
                style={{ display: "none" }}
                onChange={handleFileChange}
              />
            </label>

            <div className="text-area-container">
              <p className="or-divider">— or —</p>
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

      <MaintenanceLegend ecosystem={ecosystem} />
      <ReplaceabilityLegend />

      <Footer />
    </div>
  );
}

export default App;
