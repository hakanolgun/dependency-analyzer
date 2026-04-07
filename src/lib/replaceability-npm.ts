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
const GITHUB_REPO_IN_URL = /github\.com[/:]([^/]+)\/([^/.]+)/i;
const STRONG_NATIVE_GITHUB_LANGS = new Set([
  "Rust",
  "Go",
  "C",
  "C++",
  "Objective-C",
  "Swift",
  "Kotlin",
  "Java",
  "Zig",
  "C#",
  "Assembly",
]);
const BINARY_SHIM_MIN_BYTES = 120_000;
const BINARY_SHIM_MIN_BYTES_PER_LINE = 2000;
const unpkgMetaCache = new Map<string, Promise<UnpkgMeta | null>>();
const githubLanguageCache = new Map<string, Promise<string | null>>();
const GITHUB_LANG_MIN_INTERVAL_MS = 350;
let githubLangLastFetchAtMs = 0;
const jsdelivrMetaCache = new Map<string, Promise<JsdelivrFlatMeta | null>>();
const sourceTextCache = new Map<string, Promise<string | null>>();

interface UnpkgFileNode {
  path?: string;
  files?: UnpkgFileNode[];
}

/** Minimal shape of unpkg `?meta` JSON used by this module. */
interface UnpkgMeta {
  files?: UnpkgFileNode[];
}

interface NpmManifest {
  bin?: string | Record<string, string>;
  browser?: string;
  dependencies?: Record<string, string>;
  dist?: { unpackedSize?: number };
  main?: string;
  module?: string;
  peerDependencies?: Record<string, string>;
  repository?: string | { url?: string };
  scripts?: Record<string, string>;
  unpkg?: string;
}

interface NpmRegistryData {
  "dist-tags"?: { latest?: string };
  versions?: Record<string, NpmManifest | undefined>;
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
}

interface CalculateNpmReplaceabilityInput {
  packageName: string;
  latestVersion?: string;
  registryData: NpmRegistryData;
}

function clamp(value: number, min = 0, max = 1): number {
  return Math.max(min, Math.min(max, value));
}

function normalizeLinear(value: number, min: number, max: number): number {
  if (!Number.isFinite(value) || value <= min) return 0;
  if (value >= max) return 1;
  return (value - min) / (max - min);
}

function getLatestManifest(registryData: NpmRegistryData, latestVersion?: string): NpmManifest {
  if (latestVersion && registryData.versions?.[latestVersion]) {
    return registryData.versions[latestVersion];
  }
  const fromLatestTag = registryData.versions?.[registryData["dist-tags"]?.latest ?? ""];
  return fromLatestTag ?? {};
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

function hasNpmBinField(bin: unknown): boolean {
  if (bin == null) return false;
  if (typeof bin === "string") return bin.trim() !== "";
  if (typeof bin === "object") return Object.keys(bin as object).length > 0;
  return false;
}

function parseGitHubOwnerRepoFromManifest(repository: unknown): string {
  let raw = "";
  if (typeof repository === "string") {
    raw = repository;
  } else if (repository && typeof repository === "object" && "url" in repository) {
    const u = (repository as { url?: string }).url;
    if (typeof u === "string") raw = u;
  }
  return extractGitHubOwnerRepoFromURL(raw);
}

function extractGitHubOwnerRepoFromURL(raw: string): string {
  const t = raw.trim();
  if (/^github:/i.test(t)) {
    const tail = t
      .slice(7)
      .trim()
      .replace(/\.git$/i, "");
    const parts = tail.split("/");
    if (parts.length >= 2 && parts[0] && parts[1]) {
      return `${parts[0]}/${parts[1]}`;
    }
  }
  const s = t
    .replace(/^git\+/i, "")
    .replace(/\.git$/i, "")
    .replace(/^git:\/\//i, "https://");
  const m = s.match(GITHUB_REPO_IN_URL);
  if (!m?.[1] || !m[2]) return "";
  return `${m[1]}/${m[2]}`;
}

function binaryShimNativePresence(
  unpackedBytes: number,
  lineCount: number,
  hasBin: boolean,
): number {
  if (!hasBin || !Number.isFinite(unpackedBytes) || unpackedBytes < BINARY_SHIM_MIN_BYTES) {
    return 0;
  }
  if (lineCount < 1) {
    return 1;
  }
  if (unpackedBytes / lineCount >= BINARY_SHIM_MIN_BYTES_PER_LINE) {
    return 1;
  }
  return 0;
}

function finalizeNativeFromBinaryShim(
  base: number,
  unpackedBytes: number,
  lineCount: number,
  hasBin: boolean,
): number {
  return clamp(Math.max(base, binaryShimNativePresence(unpackedBytes, lineCount, hasBin)));
}

async function throttleGitHubLanguagesFetch(): Promise<void> {
  const now = Date.now();
  const elapsed = now - githubLangLastFetchAtMs;
  if (githubLangLastFetchAtMs > 0 && elapsed < GITHUB_LANG_MIN_INTERVAL_MS) {
    await sleep(GITHUB_LANG_MIN_INTERVAL_MS - elapsed);
  }
  githubLangLastFetchAtMs = Date.now();
}

function primaryLanguageFromLanguagesPayload(data: unknown): string | null {
  if (data === null || typeof data !== "object") return null;
  const entries = Object.entries(data as Record<string, unknown>);
  let best = "";
  let maxBytes = 0;
  for (const [lang, raw] of entries) {
    const n = typeof raw === "number" ? raw : Number(raw);
    if (!Number.isFinite(n) || n <= maxBytes) continue;
    maxBytes = n;
    best = lang;
  }
  return best.trim() || null;
}

async function fetchGitHubLanguagesOnce(url: string, init: RequestInit): Promise<Response | null> {
  try {
    return await fetch(url, init);
  } catch {
    return null;
  }
}

async function fetchGitHubPrimaryLanguage(ownerRepo: string): Promise<string | null> {
  const cached = githubLanguageCache.get(ownerRepo);
  if (cached) return cached;

  const request = (async (): Promise<string | null> => {
    const parts = ownerRepo.split("/");
    if (parts.length !== 2 || !parts[0] || !parts[1]) return null;
    const url = `https://api.github.com/repos/${encodeURIComponent(parts[0])}/${encodeURIComponent(
      parts[1],
    )}/languages`;
    const init: RequestInit = {
      headers: {
        Accept: "application/vnd.github+json",
        "User-Agent": "dependency-analyzer-replaceability",
      },
    };

    await throttleGitHubLanguagesFetch();
    let response = await fetchGitHubLanguagesOnce(url, init);
    if (response && (response.status === 429 || response.status === 403)) {
      const retryAfter = Number(response.headers.get("retry-after"));
      const backoffMs =
        Number.isFinite(retryAfter) && retryAfter > 0 ? Math.min(retryAfter * 1000, 120_000) : 2000;
      await sleep(backoffMs);
      await throttleGitHubLanguagesFetch();
      response = await fetchGitHubLanguagesOnce(url, init);
    }

    if (!response?.ok) return null;
    try {
      const data: unknown = await response.json();
      return primaryLanguageFromLanguagesPayload(data);
    } catch {
      return null;
    }
  })();

  githubLanguageCache.set(ownerRepo, request);
  return request;
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

async function fetchWithRetry(
  url: string,
  retries = 2,
  init?: RequestInit,
): Promise<Response | null> {
  for (let attempt = 0; attempt <= retries; attempt++) {
    try {
      const response = await fetch(url, init);
      if (response.status !== 429) return response;

      if (attempt === retries) return response;
      const retryAfter = Number(response.headers.get("retry-after"));
      const backoffMs =
        Number.isFinite(retryAfter) && retryAfter > 0 ? retryAfter * 1000 : 500 * (attempt + 1);
      await sleep(backoffMs);
    } catch {
      if (attempt === retries) return null;
      await sleep(300 * (attempt + 1));
    }
  }
  return null;
}

async function fetchUnpkgMeta(packageName: string, version: string): Promise<UnpkgMeta | null> {
  const cacheKey = `${packageName}@${version}`;
  const cached = unpkgMetaCache.get(cacheKey);
  if (cached) return cached;

  const request = (async (): Promise<UnpkgMeta | null> => {
    const url = `https://unpkg.com/${packageName}@${version}/?meta`;
    const response = await fetchWithRetry(url, 2);
    if (!response?.ok) return null;
    try {
      return (await response.json()) as UnpkgMeta;
    } catch {
      return null;
    }
  })();

  unpkgMetaCache.set(cacheKey, request);
  return request;
}

async function fetchJsdelivrMeta(
  packageName: string,
  version: string,
): Promise<JsdelivrFlatMeta | null> {
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
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
    } catch (_error) {
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

  if (latestVersion) {
    const unpkgMeta = await fetchUnpkgMeta(packageName, latestVersion);
    let filePaths: string[] = [];
    if (unpkgMeta?.files) {
      filePaths = collectFilePaths(unpkgMeta.files);
      nativeFromTree = hasNativeFiles(filePaths);
      shellLeakFromTree = hasShellLeakPaths(filePaths);
    }
    if (filePaths.length === 0) {
      const jsdelivrMeta = await fetchJsdelivrMeta(packageName, latestVersion);
      if (jsdelivrMeta?.files?.length) {
        filePaths = jsdelivrMeta.files
          .map((file) => file.name)
          .filter((name): name is string => typeof name === "string");
        nativeFromTree = hasNativeFiles(filePaths);
        shellLeakFromTree = hasShellLeakPaths(filePaths);
      }
    }

    const mainEntry =
      manifest?.unpkg || manifest?.browser || manifest?.module || manifest?.main || "index.js";
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
  const rawSourceLines = sourceCode ? sourceCode.split(/\r?\n/).length : 0;
  const lineCount = Math.max(rawSourceLines, 1);
  const logicKeywordCount = sourceCode ? countRegexMatches(sourceCode, LOGIC_KEYWORDS) : 0;
  const logicDensity = logicKeywordCount / lineCount;
  const hasBlackMagic = sourceCode ? BLACK_MAGIC_PATTERN.test(sourceCode) : false;
  const hasShellExec = sourceCode ? SHELL_EXEC_PATTERN.test(sourceCode) : false;

  let native = nativeFromScripts || nativeFromTree ? 1 : 0;
  const hasBin = hasNpmBinField(manifest?.bin);
  native = finalizeNativeFromBinaryShim(native, unpackedSize, rawSourceLines, hasBin);
  const ghRepo = parseGitHubOwnerRepoFromManifest(manifest?.repository);
  if (ghRepo) {
    const lang = await fetchGitHubPrimaryLanguage(ghRepo);
    if (lang && STRONG_NATIVE_GITHUB_LANGS.has(lang)) {
      native = Math.max(native, 1);
    }
  }
  native = clamp(native);

  const metrics: NpmReplaceabilityMetrics = {
    native,
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

  return {
    normalized,
    score: Math.round(normalized * 100),
    metrics,
  };
}
