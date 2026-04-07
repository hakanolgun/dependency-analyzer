export interface GoModuleResult {
  name: string;
  currentVersion: string;
  latestVersion?: string;
  replaceability?: number;
  lastUpdateDate?: string;
  timeSinceLastUpdate?: string;
  isMaintained?: "yes" | "unlikely" | "no";
  repoUrl?: string;
  error?: string;
  status: "pending" | "loading" | "success" | "error";
}

export interface GoModParseResult {
  moduleName: string;
  goVersion: string;
  dependencies: { path: string; version: string }[];
}

function formatTimeSince(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();

  let diffInDays = Math.floor((now.getTime() - date.getTime()) / (1000 * 3600 * 24));

  if (diffInDays < 0) diffInDays = 0;

  if (diffInDays === 0) return "Today";
  if (diffInDays < 30) return `${diffInDays} day${diffInDays > 1 ? "s" : ""} ago`;

  const diffInMonths = Math.floor(diffInDays / 30);
  if (diffInMonths < 12) return `${diffInMonths} month${diffInMonths > 1 ? "s" : ""} ago`;

  const diffInYears = Math.floor(diffInDays / 365);
  return `${diffInYears} year${diffInYears > 1 ? "s" : ""} ago`;
}

function getMockReplaceabilityScore(seed: string): number {
  let hash = 0;
  for (let i = 0; i < seed.length; i++) {
    hash = (hash * 37 + seed.charCodeAt(i)) % 1000;
  }
  return 20 + (hash % 71);
}

/**
 * Encode a Go module path for use in proxy.golang.org URLs.
 * Uppercase letters must be escaped with a '!' prefix and lowercased.
 * e.g. "github.com/Azure/go-sdk" → "github.com/!azure/go-sdk"
 */
export function encodeModulePath(modulePath: string): string {
  return modulePath.replace(/[A-Z]/g, (ch) => `!${ch.toLowerCase()}`);
}

/**
 * Parse a go.mod file content and extract direct dependencies.
 * Skips indirect dependencies (lines with "// indirect").
 */
export function parseGoMod(content: string): GoModParseResult | null {
  const lines = content.split("\n");

  // Extract module name
  const moduleLine = lines.find((l) => l.trim().startsWith("module "));
  const moduleName = moduleLine ? moduleLine.trim().replace("module ", "").trim() : "";

  // Extract go version
  const goVersionLine = lines.find((l) => l.trim().startsWith("go "));
  const goVersion = goVersionLine ? goVersionLine.trim().replace("go ", "").trim() : "";

  const dependencies: { path: string; version: string }[] = [];

  let inRequireBlock = false;

  for (const line of lines) {
    const trimmed = line.trim();

    // Skip indirect dependencies
    if (trimmed.includes("// indirect")) continue;

    // Single-line require: require github.com/pkg/errors v0.9.1
    if (trimmed.startsWith("require ") && !trimmed.includes("(")) {
      const parts = trimmed.replace("require ", "").trim().split(/\s+/);
      if (parts.length >= 2) {
        dependencies.push({ path: parts[0], version: parts[1] });
      }
      continue;
    }

    // Block require start
    if (trimmed === "require (" || trimmed.startsWith("require (")) {
      inRequireBlock = true;
      continue;
    }

    // Block require end
    if (inRequireBlock && trimmed === ")") {
      inRequireBlock = false;
      continue;
    }

    // Inside require block
    if (inRequireBlock && trimmed.length > 0 && !trimmed.startsWith("//")) {
      const parts = trimmed.split(/\s+/);
      if (parts.length >= 2) {
        dependencies.push({ path: parts[0], version: parts[1] });
      }
    }
  }

  if (dependencies.length === 0 && !moduleName) return null;

  return { moduleName, goVersion, dependencies };
}

/**
 * Fetch module data from the Go module proxy (proxy.golang.org).
 */
export async function fetchGoModuleData(
  name: string,
  currentVersion: string,
): Promise<GoModuleResult> {
  const result: GoModuleResult = {
    name,
    currentVersion,
    replaceability: getMockReplaceabilityScore(name),
    status: "loading",
  };

  try {
    const encodedPath = encodeModulePath(name);

    // Fetch latest version info
    const latestRes = await fetch(`https://proxy.golang.org/${encodedPath}/@latest`);
    if (!latestRes.ok) {
      throw new Error(`Module not found (${latestRes.status})`);
    }
    const latestData = await latestRes.json();

    result.latestVersion = latestData.Version;

    if (latestData.Time) {
      result.lastUpdateDate = latestData.Time;
      result.timeSinceLastUpdate = formatTimeSince(latestData.Time);
    }

    // Extract repo URL from Origin if available
    if (latestData.Origin?.URL) {
      result.repoUrl = latestData.Origin.URL;
    }

    // Maintained heuristic: Go modules use 3 years (npm uses 2).
    if (latestData.Time) {
      const lastUpdated = new Date(latestData.Time).getTime();
      const threeYearsMs = 3 * 365 * 24 * 60 * 60 * 1000;
      const threshold = Date.now() - threeYearsMs;

      if (lastUpdated < threshold) {
        result.isMaintained = "unlikely";
      } else {
        result.isMaintained = "yes";
      }
    }

    result.status = "success";
    return result;
  } catch (err: unknown) {
    result.status = "error";
    result.error = err instanceof Error ? err.message : String(err);
    return result;
  }
}
