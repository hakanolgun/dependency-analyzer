import { Circle, TriangleAlert, OctagonX } from "lucide-react";
import { CopyCommand, npxCommand } from "./CopyCommand";


export function ReplaceabilityLegend() {
  return (
    <div className="maintenance-legend-section">
      <div className="legend-container">
        <h3 className="legend-title">Replaceability Score</h3>
        <p className="cli-command-description">
          Replaceability Score measures how difficult it would be to replace a particular dependency
          in your project. A higher score indicates that removing or replacing the dependency would
          require significant effort and could introduce risk.
        </p>
        <p className="cli-command-description">
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
          <h4 className="legend-title">How to use?</h4>
          <p className="cli-command-description">
            Run the npx command below to analyze your project's dependencies. A detailed HTML report
            will automatically open in your browser upon completion.
          </p>
          <p className="cli-command-description">
            Look for packages with low maintenance or high replaceability scores. These are your
            primary targets for potential replacement.
          </p>

          <CopyCommand command={npxCommand} />
        </div>
      </div>
    </div>
  );
}
