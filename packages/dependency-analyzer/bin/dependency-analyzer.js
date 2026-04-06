#!/usr/bin/env node
import { existsSync } from "node:fs";
import { mkdir, chmod } from "node:fs/promises";
import path from "node:path";
import { spawn, spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const packageRoot = path.resolve(__dirname, "..");

const platformMap = {
  win32: "windows",
  darwin: "darwin",
  linux: "linux",
};

const archMap = {
  x64: "amd64",
  arm64: "arm64",
};

const platform = platformMap[process.platform] ?? process.platform;
const arch = archMap[process.arch] ?? process.arch;
const ext = process.platform === "win32" ? ".exe" : "";
const binaryName = `dependency-analyzer-${platform}-${arch}${ext}`;
const binaryPath = path.join(packageRoot, "dist", binaryName);

async function ensureBinaryExecutable(targetPath) {
  if (platform !== "win32") {
    await chmod(targetPath, 0o755);
  }
}

function runBinary(targetPath) {
  const child = spawn(targetPath, process.argv.slice(2), {
    stdio: "inherit",
  });
  child.on("exit", (code) => process.exit(code ?? 1));
}

function tryBuildFromSource() {
  // Fallback for local development or source installs.
  const repoRoot = path.resolve(packageRoot, "..", "..");
  const goModuleDir = path.join(repoRoot, "cli-go");
  const sourceMain = path.join(goModuleDir, "cmd", "dependency-analyzer", "main.go");
  if (!existsSync(sourceMain)) {
    return null;
  }

  const outDir = path.join(packageRoot, "dist");
  const outFile = binaryPath;
  return mkdir(outDir, { recursive: true })
    .then(() => {
      const result = spawnSync("go", ["build", "-o", outFile, "./cmd/dependency-analyzer"], {
        cwd: goModuleDir,
        stdio: "inherit",
      });
      if (result.status !== 0) {
        return null;
      }
      return outFile;
    })
    .catch(() => null);
}

const start = async () => {
  if (existsSync(binaryPath)) {
    await ensureBinaryExecutable(binaryPath);
    runBinary(binaryPath);
    return;
  }

  const builtPath = await tryBuildFromSource();
  if (builtPath && existsSync(builtPath)) {
    await ensureBinaryExecutable(builtPath);
    runBinary(builtPath);
    return;
  }

  console.error(
    "dependency-analyzer: no bundled binary found and fallback Go build failed. Reinstall package or install Go toolchain.",
  );
  process.exit(1);
};

start();
