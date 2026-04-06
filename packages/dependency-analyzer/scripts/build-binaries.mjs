import { mkdir, chmod } from "node:fs/promises";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const packageRoot = path.resolve(__dirname, "..");
const repoRoot = path.resolve(packageRoot, "..", "..");
const goModuleDir = path.join(repoRoot, "cli-go");
const outDir = path.join(packageRoot, "dist");

const targets = [
  { goos: "darwin", goarch: "arm64" },
  { goos: "darwin", goarch: "amd64" },
  { goos: "linux", goarch: "amd64" },
  { goos: "linux", goarch: "arm64" },
  { goos: "windows", goarch: "amd64" },
];

await mkdir(outDir, { recursive: true });

for (const target of targets) {
  const isWindows = target.goos === "windows";
  const outName = `dependency-analyzer-${target.goos}-${target.goarch}${isWindows ? ".exe" : ""}`;
  const outPath = path.join(outDir, outName);

  const env = {
    ...process.env,
    CGO_ENABLED: "0",
    GOOS: target.goos,
    GOARCH: target.goarch,
  };

  process.stdout.write(`Building ${outName}...\n`);
  const res = spawnSync("go", ["build", "-o", outPath, "./cmd/dependency-analyzer"], {
    cwd: goModuleDir,
    env,
    stdio: "inherit",
  });

  if (res.status !== 0) {
    process.exit(res.status ?? 1);
  }

  if (!isWindows) {
    await chmod(outPath, 0o755);
  }
}

process.stdout.write("All binaries built.\n");
