# 🛡️ Dependency Analyzer

**Dependency-Analyzer** is a powerful dependency analysis tool designed to evaluate the "Replaceability" of your project's dependencies. It doesn't just list your packages; it deep-dives into their source code and metadata to calculate exactly how much effort would be required to replace them.

Ships as a **high-performance Go CLI**, with an optional **npm** wrapper for installation (`@vinean/dependency-analyzer`).

---

## 🚀 Key Pillars of Replaceability

Dependency Analyzer evaluates every dependency across five critical metrics to generate a normalized **Replaceability Cost (0-100)**. A higher score indicates a dependency that is more "locked-in" and costly to replace.

### 1. 🏗️ Native Presence (40%)

Detects native code signals (C++, `node-gyp`, `cgo`, `syscall`, `unsafe`). Native dependencies often require specific build environments and are significantly harder to port or replace with pure-logic alternatives.

### 2. 📦 Code Volume (10%)

Analyzes the physical size and source lines of code (SLOC). While not a direct measure of complexity, massive packages represent a larger "surface area" of logic that your project might be relying on.

### 3. 🌐 API Surface (10%)

Measures the breadth of the public interface.

- **NPM**: Counts exports, classes, and methods.
- **Go**: Evaluates exported functions, structs, interfaces, and anonymous nesting depth.
- **Structural Penalty**: Higher nesting levels (Max Brace Depth) increase the score.

### 4. 🪢 Entanglement (15%)

Analyzes dependency chains.

- Tracks direct and peer dependencies.
- Heuristically estimates **Transitive Depth**.
- Detects "Shell Leaks" (imports of `child_process`, `os/exec`, etc.) that suggest deep OS-level integration.

### 5. 🧠 Logic Complexity (25%)

A proxy for cognitive complexity.

- **NPM**: Decision point density (if/else, switch, catch).
- **Go**: Measures concurrency features (`goroutines`, `channels`, `defer`) alongside cyclomatic proxies.
- **Confidence Modifier**: High test-file counts (coverage) slightly reduce this score, as well-tested code is easier to refactor/replace.

---

## 🛠️ Supported Ecosystems

| Ecosystem         | Detected File  | Analysis Level                                                              |
| :---------------- | :------------- | :-------------------------------------------------------------------------- |
| **NPM / Node.js** | `package.json` | Local `node_modules` scan first; optional tarball fetch + registry metadata |
| **Go Modules**    | `go.mod`       | Proxy zip download + Source code parsing                                    |

---

## 💻 CLI Usage

The CLI is written in Go for speed, allowing it to parse thousands of files in milliseconds.

### Installation

```bash
# Using npm
npx @vinean/dependency-analyzer

# Or build from source
cd cli-go
go build -o dependency-analyzer ./cmd/dependency-analyzer
```

### Commands

```bash
# Scan current directory and open report
npx @vinean/dependency-analyzer

# Scan specific project without opening browser
npx @vinean/dependency-analyzer --project ./my-cool-app --open=false

# Output raw JSON summary to stdout
npx @vinean/dependency-analyzer --json
```

### NPM analysis (local install vs. registry)

For JavaScript projects the CLI **prefers an existing install**: it reads each direct dependency from `node_modules` (including pnpm-style symlinks, which are resolved before scanning).

If a package is **not** on disk (no install, Yarn Plug’n’Play without `node_modules`, and so on), it can **fetch that package’s tarball from the npm registry**, extract it to a temporary directory, run the same code metrics, then delete the temp folder. Resolved versions come from, in order: `package-lock.json`, `pnpm-lock.yaml`, `yarn.lock` (classic), or an **exact** version string in `package.json` (ranges like `^1.0.0` are not enough without a lockfile).

| Flag            | Effect                                                                                                                                                                                        |
| :-------------- | :-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `--no-ghost`    | Do not download tarballs; only analyze what is present under `node_modules`. Useful for air-gapped CI or strict “no fetch” policies.                                                          |
| `--no-registry` | Skip npm registry **metadata** (downloads, maintenance hints, React Native directory, etc.). Code analysis still runs; ghost tarball fetch still runs when needed unless `--no-ghost` is set. |

```bash
# Fail if dependencies are not installed (no network tarball fetch)
npx @vinean/dependency-analyzer --no-ghost

# Analyze from disk / tarballs only; no registry API calls
npx @vinean/dependency-analyzer --no-registry

# Combine both: strictly local node_modules, no registry calls
npx @vinean/dependency-analyzer --no-ghost --no-registry
```

---

## 📂 Project Structure

- `cli-go/`: Core analysis engine and HTML report generation (Go).
- `packages/dependency-analyzer/`: Node.js wrapper and cross-platform binary build for the published npm package.

---

## 📜 License

Distributed under the GNU AFFERO GENERAL PUBLIC LICENSE v3.0. See `LICENSE.md` for more information.

---
