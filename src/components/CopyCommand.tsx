import { useState } from "react";
import { Copy, Check } from "lucide-react";

export const npxCommand = "npx @vinean/dependency-analyzer";

export function CopyCommand({ command }: { command: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(command);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        gap: "1rem",
      }}>
      <div className="cli-command-container" onClick={handleCopy}>
        <p className="cli-command">{command}</p>
        <div className="copy-icon-wrapper">
          {copied ? <Check size={18} className="copied-success" /> : <Copy size={18} />}
        </div>
      </div>
      {copied && <span className="copy-hint copied-success">Copied to clipboard!</span>}
    </div>
  );
}
