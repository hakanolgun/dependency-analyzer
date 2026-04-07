const WEIGHTS = {
  native: 0.4,
  volume: 0.1,
  apiSurface: 0.1,
  entanglement: 0.15,
  logicComplexity: 0.25,
} as const;

const MIN_VOLUME_BYTES = 100 * 1024;
const MAX_VOLUME_BYTES = 10 * 1024 * 1024;
const NATIVE_FILE_EXTENSIONS = [
  ".cpp",
  ".h",
  ".cc",
  ".swift",
  ".kt",
  ".java",
  ".node",
  ".wasm",
  ".rs",
];
const LOGIC_KEYWORDS = /\b(if|else|switch|map|filter|reduce|catch|async)\b/g;
const EXPORT_PATTERN = /export\s+(const|function|class|interface|type)\b/g;
const BLACK_MAGIC_PATTERN = /(eval\s*\(|new\s+Function\s*\(|Buffer\.from\s*\()/g;
const SHELL_IMPORT_PATTERN = /\b(child_process|execa)\b/g;
const SHELL_EXEC_PATTERN = /(\.spawn\s*\(|\.exec\s*\()/g;
const unpkgMetaCache = new Map<string, Promise<any | null>>();
const jsdelivrMetaCache = new Map<string, Promise<any | null>>();
const sourceTextCache = new Map<string, Promise<string | null>>();

interface UnpkgFileNode {
  path?: string;
  files?: UnpkgFileNode[];
}

interface JsdelivrFlatMeta {
  files?: Array<{ name?: string }>;
}

export interface NpmReplaceabilityMetrics {
  native: number;
  volume: number;
  apiSurface: number;
  entanglement: number;
  logicComplexity: number;
}

export interface NpmReplaceabilityResult {
  normalized: number;
  score: number;
  metrics: NpmReplaceabilityMetrics;
  confidence: "high" | "medium" | "low";
}

interface CalculateNpmReplaceabilityInput {
  packageName: string;
  latestVersion?: string;
  registryData: any;
}

function clamp(value: number, min = 0, max = 1): number {
  return Math.max(min, Math.min(max, value));
}

function normalizeLinear(value: number, min: number, max: number): number {
  if (!Number.isFinite(value) || value <= min) return 0;
  if (value >= max) return 1;
  return (value - min) / (max - min);
}

function getLatestManifest(registryData: any, latestVersion?: string): any {
  if (latestVersion && registryData?.versions?.[latestVersion]) {
    return registryData.versions[latestVersion];
  }
  return registryData?.versions?.[registryData?.["dist-tags"]?.latest] || {};
}

function normalizeMainFilePath(mainFile: string | undefined): string {
  if (!mainFile) return "index.js";
  if (mainFile.startsWith("./")) return mainFile.slice(2);
  if (mainFile.startsWith("/")) return mainFile.slice(1);
  return mainFile;
}

function ensureLeadingSlash(path: string): string {
  return path.startsWith("/") ? path : `/${path}`;
}

function buildCandidateMainPaths(mainPath: string, availablePaths: string[]): string[] {
  const normalized = ensureLeadingSlash(normalizeMainFilePath(mainPath));
  const hasFileExtension = /\.[a-z0-9]+$/i.test(normalized);
  const candidates = new Set<string>([
    normalized,
    `${normalized}.js`,
    `${normalized}.mjs`,
    `${normalized}.cjs`,
    `${normalized}.ts`,
    `${normalized}.tsx`,
    `${normalized}/index.js`,
    `${normalized}/index.mjs`,
    `${normalized}/index.cjs`,
    `${normalized}/index.ts`,
    `${normalized}/index.tsx`,
  ]);

  if (hasFileExtension) {
    candidates.delete(`${normalized}/index.js`);
    candidates.delete(`${normalized}/index.mjs`);
    candidates.delete(`${normalized}/index.cjs`);
    candidates.delete(`${normalized}/index.ts`);
    candidates.delete(`${normalized}/index.tsx`);
  }

  if (availablePaths.length > 0) {
    const lower = normalized.toLowerCase();
    const exactMatch = availablePaths.find((p) => p.toLowerCase() === lower);
    if (exactMatch) candidates.add(ensureLeadingSlash(exactMatch));

    const related = availablePaths.filter((p) => p.toLowerCase().startsWith(lower));
    for (const path of related.slice(0, 5)) {
      if (/\.(js|mjs|cjs|ts|tsx)$/.test(path)) {
        candidates.add(ensureLeadingSlash(path));
      }
    }
  }

  return [...candidates];
}

function countRegexMatches(content: string, pattern: RegExp): number {
  const matches = content.match(pattern);
  return matches ? matches.length : 0;
}

function collectFilePaths(nodes: UnpkgFileNode[] | undefined, acc: string[] = []): string[] {
  if (!nodes) return acc;
  for (const node of nodes) {
    if (typeof node.path === "string") {
      acc.push(node.path);
    }
    if (node.files?.length) {
      collectFilePaths(node.files, acc);
    }
  }
  return acc;
}

function hasNativeFiles(paths: string[]): boolean {
  const lowerPaths = paths.map((p) => p.toLowerCase());
  return lowerPaths.some((path) => NATIVE_FILE_EXTENSIONS.some((ext) => path.endsWith(ext)));
}

function hasShellLeakPaths(paths: string[]): boolean {
  const lowerPaths = paths.map((p) => p.toLowerCase());
  return lowerPaths.some((path) => path.includes("/bin/") || path.endsWith("cli.js"));
}

function computeWeightedScore(metrics: NpmReplaceabilityMetrics): number {
  return (
    metrics.native * WEIGHTS.native +
    metrics.volume * WEIGHTS.volume +
    metrics.apiSurface * WEIGHTS.apiSurface +
    metrics.entanglement * WEIGHTS.entanglement +
    metrics.logicComplexity * WEIGHTS.logicComplexity
  );
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function fetchWithRetry(url: string, retries = 2): Promise<Response | null> {
  for (let attempt = 0; attempt <= retries; attempt++) {
    try {
      const response = await fetch(url);
      if (response.status !== 429) return response;

      if (attempt === retries) return response;
      const retryAfter = Number(response.headers.get("retry-after"));
      const backoffMs = Number.isFinite(retryAfter) && retryAfter > 0
        ? retryAfter * 1000
        : 500 * (attempt + 1);
      await sleep(backoffMs);
    } catch {
      if (attempt === retries) return null;
      await sleep(300 * (attempt + 1));
    }
  }
  return null;
}

async function fetchUnpkgMeta(packageName: string, version: string): Promise<any | null> {
  const cacheKey = `${packageName}@${version}`;
  const cached = unpkgMetaCache.get(cacheKey);
  if (cached) return cached;

  const request = (async () => {
    const url = `https://unpkg.com/${packageName}@${version}/?meta`;
    const response = await fetchWithRetry(url, 2);
    if (!response?.ok) return null;
    try {
      return await response.json();
    } catch {
      return null;
    }
  })();

  unpkgMetaCache.set(cacheKey, request);
  return request;
}

async function fetchJsdelivrMeta(packageName: string, version: string): Promise<JsdelivrFlatMeta | null> {
  const cacheKey = `${packageName}@${version}`;
  const cached = jsdelivrMetaCache.get(cacheKey);
  if (cached) return cached;

  const request = (async () => {
    const url = `https://data.jsdelivr.com/v1/package/npm/${packageName}@${version}/flat`;
    const response = await fetchWithRetry(url, 2);
    if (!response?.ok) return null;
    try {
      return (await response.json()) as JsdelivrFlatMeta;
    } catch {
      return null;
    }
  })();

  jsdelivrMetaCache.set(cacheKey, request);
  return request;
}

async function fetchSourceFromCdn(url: string): Promise<string | null> {
  const cached = sourceTextCache.get(url);
  if (cached) return cached;

  const request = (async () => {
    try {
      const response = await fetch(url);
      if (!response.ok) {
        return null;
      }
      return await response.text();
    } catch (error) {
      return null;
    }
  })();

  sourceTextCache.set(url, request);
  return request;
}

async function fetchMainEntrySource(
  packageName: string,
  version: string,
  mainPath: string,
  availablePaths: string[],
): Promise<string | null> {
  const candidatePaths = buildCandidateMainPaths(mainPath, availablePaths);
  for (const candidatePath of candidatePaths) {
    const unpkgUrl = `https://unpkg.com/${packageName}@${version}${candidatePath}`;
    const unpkgSource = await fetchSourceFromCdn(unpkgUrl);
    if (unpkgSource) return unpkgSource;

    const jsdelivrUrl = `https://cdn.jsdelivr.net/npm/${packageName}@${version}${candidatePath}`;
    const jsdelivrSource = await fetchSourceFromCdn(jsdelivrUrl);
    if (jsdelivrSource) return jsdelivrSource;
  }
  return null;
}

export async function calculateNpmReplaceability({
  packageName,
  latestVersion,
  registryData,
}: CalculateNpmReplaceabilityInput): Promise<NpmReplaceabilityResult> {
  const manifest = getLatestManifest(registryData, latestVersion);

  const dependencies = Object.keys(manifest?.dependencies || {}).length;
  const peerDependencies = Object.keys(manifest?.peerDependencies || {}).length;
  const unpackedSize = Number(manifest?.dist?.unpackedSize || 0);
  const scripts = manifest?.scripts || {};

  const installScript = `${scripts.install || ""} ${scripts.postinstall || ""}`.toLowerCase();
  const nativeFromScripts =
    installScript.includes("node-gyp") || installScript.includes("cmake-js");

  let nativeFromTree = false;
  let shellLeakFromTree = false;
  let shellLeakFromSource = false;
  let sourceCode = "";
  let hasTreeData = false;

  if (latestVersion) {
    const unpkgMeta = await fetchUnpkgMeta(packageName, latestVersion);
    let filePaths: string[] = [];
    if (unpkgMeta?.files) {
      filePaths = collectFilePaths(unpkgMeta.files);
      hasTreeData = filePaths.length > 0;
      nativeFromTree = hasNativeFiles(filePaths);
      shellLeakFromTree = hasShellLeakPaths(filePaths);
    }
    if (filePaths.length === 0) {
      const jsdelivrMeta = await fetchJsdelivrMeta(packageName, latestVersion);
      if (jsdelivrMeta?.files?.length) {
        filePaths = jsdelivrMeta.files
          .map((file) => file.name)
          .filter((name): name is string => typeof name === "string");
        hasTreeData = filePaths.length > 0;
        nativeFromTree = hasNativeFiles(filePaths);
        shellLeakFromTree = hasShellLeakPaths(filePaths);
      }
    }

    const mainEntry =
      manifest?.unpkg ||
      manifest?.browser ||
      manifest?.module ||
      manifest?.main ||
      "index.js";
    const resolvedMain = typeof mainEntry === "string" ? mainEntry : "index.js";
    const entrySource = await fetchMainEntrySource(
      packageName,
      latestVersion,
      resolvedMain,
      filePaths,
    );
    if (entrySource) {
      sourceCode = entrySource;
      shellLeakFromSource = SHELL_IMPORT_PATTERN.test(sourceCode);
    }
  }

  const exportCount = sourceCode ? countRegexMatches(sourceCode, EXPORT_PATTERN) : 0;
  const lineCount = sourceCode ? sourceCode.split(/\r?\n/).length : 1;
  const logicKeywordCount = sourceCode ? countRegexMatches(sourceCode, LOGIC_KEYWORDS) : 0;
  const logicDensity = logicKeywordCount / Math.max(lineCount, 1);
  const hasBlackMagic = sourceCode ? BLACK_MAGIC_PATTERN.test(sourceCode) : false;
  const hasShellExec = sourceCode ? SHELL_EXEC_PATTERN.test(sourceCode) : false;

  const metrics: NpmReplaceabilityMetrics = {
    native: nativeFromScripts || nativeFromTree ? 1 : 0,
    volume: normalizeLinear(unpackedSize, MIN_VOLUME_BYTES, MAX_VOLUME_BYTES),
    apiSurface: clamp(exportCount / 50),
    entanglement: clamp((dependencies + peerDependencies * 2) / 40),
    logicComplexity: clamp(logicDensity * 8),
  };

  if (shellLeakFromTree || shellLeakFromSource) {
    metrics.entanglement = clamp(metrics.entanglement + 0.1);
  }
  if (hasShellExec) {
    metrics.logicComplexity = clamp(metrics.logicComplexity + 0.1);
  }
  if (hasBlackMagic) {
    metrics.logicComplexity = clamp(metrics.logicComplexity + 0.1);
  }

  const normalized = clamp(computeWeightedScore(metrics));
  const confidence: "high" | "medium" | "low" =
    sourceCode && hasTreeData ? "high" : sourceCode || hasTreeData ? "medium" : "low";

  return {
    normalized,
    score: Math.round(normalized * 100),
    metrics,
    confidence,
  };
}
