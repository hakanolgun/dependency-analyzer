import { useState } from "react";
import { Circle, TriangleAlert, OctagonX, Copy, Check } from "lucide-react";

export function ReplaceabilityLegend() {
  const [copied, setCopied] = useState(false);
  const command = "npx @vinean/dependency-analyzer";

  const handleCopy = () => {
    navigator.clipboard.writeText(command);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };
  return (
    <div className="maintenance-legend-section">
      <div className="legend-container">
        <h3 className="legend-title">Replaceability Score</h3>
        <p className="legend-description">
          Replaceability Score measures how difficult it would be to replace a particular dependency in your project. A higher score indicates that removing or replacing the dependency would require significant effort and could introduce risk.
          If the score is low, it suggests that the dependency is easier to replace—either by implementing the functionality yourself or by using an LLM to generate an alternative solution.
        </p>
        <p className="legend-description">
          Replaceability ≠ Complexity. Replaceability ⊇ Complexity.
        </p>
        <div className="legend-grid">
          <div className="legend-item">
            <div className="legend-badge badge-yes">
              <Circle size={16} />
              <span>Easy (0-30)</span>
            </div>
            <p className="legend-description">
              Mostly small and straightforward libraries with low complexity.
            </p>
          </div>

          <div className="legend-item">
            <div className="legend-badge badge-unlikely">
              <TriangleAlert size={16} />
              <span>Medium (31-70)</span>
            </div>
            <p className="legend-description">
              Moderate logic and dependency coupling. Replacement is possible with focused effort.
            </p>
          </div>

          <div className="legend-item">
            <div className="legend-badge badge-no">
              <OctagonX size={16} />
              <span>Hard (71-100)</span>
            </div>
            <p className="legend-description">
              Native bindings, broad API surface, or complex internals make replacement difficult.
            </p>
          </div>
        </div>

        <div className="cli-command-wrapper">
          <p className="cli-command-description">
            To calculate the replaceability score of your dependencies, run the CLI analyzer from your project’s root directory.
          </p>

          <div style={{ display: "flex", alignItems: "center", justifyContent: "center", gap: "1rem" }}>
          <div className="cli-command-container" onClick={handleCopy}>
            <p className="cli-command">
              {command}
            </p>
            <div className="copy-icon-wrapper">
              {copied ? (
                <Check size={18} className="copied-success" />
              ) : (
                <Copy size={18} />
              )}
            </div>
          </div>
          {copied && <span className="copy-hint copied-success">Copied to clipboard!</span>}
        </div>
          </div>
      </div>
    </div>
  );
}
