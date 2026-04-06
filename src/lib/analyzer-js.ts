export interface PackageResult {
  name: string;
  currentVersion: string;
  latestVersion?: string;
  weeklyDownloads?: number;
  lastUpdateDate?: string;
  timeSinceLastUpdate?: string;
  isReactNative: boolean;
  newArchitecture?: boolean;
  isMaintained?: "yes" | "unlikely" | "no";
  repoUrl?: string;
  error?: string;
  status: "pending" | "loading" | "success" | "error";
}

export interface ParseResult {
  dependencies: Record<string, string>;
  devDependencies: Record<string, string>;
}

// Implement a queue specifically for the downloads API to enforce 2 req/sec
let downloadQueue: Promise<void> = Promise.resolve();

// Format the difference in time
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

export function parsePackageJson(content: string): ParseResult | null {
  try {
    const data = JSON.parse(content);
    return {
      dependencies: data.dependencies || {},
      devDependencies: data.devDependencies || {},
    };
  } catch (e) {
    console.log("e", e);
    return null;
  }
}

export async function fetchPackageData(
  name: string,
  currentVersion: string,
  isReactNativeProject: boolean,
): Promise<PackageResult> {
  const result: PackageResult = {
    name,
    currentVersion,
    isReactNative: false,
    status: "loading",
  };

  try {
    // 1. Fetch registry info
    const regRes = await fetch(`https://registry.npmjs.org/${name}`);
    if (!regRes.ok) throw new Error("Not found");
    const regData = await regRes.json();

    result.latestVersion = regData["dist-tags"]?.latest;

    const time = regData.time || {};
    const lastUpdateDate = time[result.latestVersion!] || time.modified;
    if (lastUpdateDate) {
      result.lastUpdateDate = lastUpdateDate;
      result.timeSinceLastUpdate = formatTimeSince(lastUpdateDate);
    }

    // Extract repo URL or homepage
    if (regData.repository?.url) {
      // Clean up git URLs
      result.repoUrl = regData.repository.url
        .replace(/^git\+/, "")
        .replace(/\.git$/, "")
        .replace(/^git:\/\//, "https://");
    } else if (regData.homepage) {
      result.repoUrl = regData.homepage;
    } else {
      // Fallback to npmjs.com
      result.repoUrl = `https://www.npmjs.com/package/${name}`;
    }

    // 2. Maintained logic
    const latest = result.latestVersion;
    if (latest) {
      const lastUpdated = time[latest];
      const deprecationMessage = regData.versions?.[latest]?.deprecated;

      const isDeprecated = !!deprecationMessage;
      // Heuristic: If no updates in 2 years and not version 1.0.0+ yet,
      // it might be abandoned.
      const isLikelyAbandoned = lastUpdated
        ? new Date(lastUpdated).getTime() < Date.now() - 63072000000
        : false;

      if (isDeprecated) {
        result.isMaintained = "no";
      } else if (isLikelyAbandoned) {
        result.isMaintained = "unlikely";
      } else {
        result.isMaintained = "yes";
      }
    }

    // 4. Fetch downloads (throttled to max 2 req/sec to prevent 429 Too Many Requests)
    result.weeklyDownloads = await new Promise<number | undefined>((resolve) => {
      downloadQueue = downloadQueue.then(async () => {
        try {
          const dlRes = await fetch(`https://api.npmjs.org/downloads/point/last-week/${name}`);
          if (dlRes.ok) {
            const dlData = await dlRes.json();
            resolve(dlData.downloads);
          } else {
            resolve(undefined);
          }
        } catch {
          resolve(undefined);
        }
        await new Promise((r) => setTimeout(r, 350));
      });
    });

    // 5. React Native check
    if (isReactNativeProject) {
      // Default to true assuming packages not found in directory are pure JS
      result.newArchitecture = true;

      const rnRes = await fetch(`https://reactnative.directory/api/libraries?search=${name}`);
      if (rnRes.ok) {
        const rnData = await rnRes.json();
        // Exact match
        const exactMatch = rnData?.libraries?.find(
          (lib: { npmPkg: string }) => lib.npmPkg === name,
        );
        if (exactMatch) {
          result.isReactNative = true;
          // New Architecture is supported if it's explicitly marked true,
          // or if the library doesn't have native code at all (pure JS),
          // or if it's an Expo module (which inherently supports it).
          result.newArchitecture =
            !!exactMatch.newArchitecture ||
            exactMatch.github?.newArchitecture === true ||
            exactMatch.github?.hasNativeCode === false ||
            exactMatch.github?.moduleType === "expo";

          // Override maintained status if React Native directory explicitly marks it as unmaintained
          if (exactMatch.unmaintained === true) {
            result.isMaintained = "no";
          }
        }
      }
    }

    result.status = "success";
    return result;
  } catch (err) {
    result.status = "error";
    result.error = err instanceof Error ? err.message : "Unknown error";
    return result;
  }
}
