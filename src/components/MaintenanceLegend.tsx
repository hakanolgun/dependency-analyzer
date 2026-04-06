import { CheckCircle2, AlertTriangle, XCircle } from "lucide-react";

export function MaintenanceLegend() {
  return (
    <div className="maintenance-legend-section">
      <div className="legend-container">
        <h3 className="legend-title">Maintenance Status Key</h3>
        <div className="legend-grid">
          <div className="legend-item">
            <div className="legend-badge badge-yes">
              <CheckCircle2 size={16} />
              <span>Yes</span>
            </div>
            <p className="legend-description">
              The package is active, has recent updates, and is not officially deprecated.
            </p>
          </div>
          
          <div className="legend-item">
            <div className="legend-badge badge-unlikely">
              <AlertTriangle size={16} />
              <span>Unlikely</span>
            </div>
            <p className="legend-description">
              No updates in the last 2 years. The package might be abandoned.
            </p>
          </div>
          
          <div className="legend-item">
            <div className="legend-badge badge-no">
              <XCircle size={16} />
              <span>No</span>
            </div>
            <p className="legend-description">
              Explicitly marked as deprecated on NPM or marked as unmaintained by the community.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
