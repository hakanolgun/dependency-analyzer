# 🛡️ @vinean/dependency-analyzer

A high-performance CLI for analyzing the **Replaceability**, **Maintenance**, and **Health** of your project's dependencies. Supports both **NPM packages** and **Go Modules**.

## 🚀 Usage

Run it directly via `npx` in your project root directory:

```bash
npx @vinean/dependency-analyzer
```

Or install it globally:

```bash
npm install -g @vinean/dependency-analyzer
@vinean/dependency-analyzer --project ./my-cool-project
```

## ⚙️ Options

- `--project <path>`: Path to the project root (default: current directory).
- `--ecosystem <type>`: Force ecosystem detection (`npm` or `go`). Auto-detected by default.
- `--open=false`: Disable auto-opening the generated HTML report.
- `--json`: Print the raw analysis summary (JSON) to stdout.
- `--no-ghost` (**npm only**): Do not fetch package tarballs from the registry when `node_modules` is missing. Analysis uses only what is installed locally.
- `--no-registry` (**npm only**): Skip npm registry metadata (weekly downloads, maintenance heuristics, React Native directory). Does not disable tarball fetch for code analysis; use `--no-ghost` for that.

### NPM: local install first, registry fallback

By default, dependencies are read from `node_modules` when present. If a direct dependency is missing on disk, the tool downloads its **exact** version from the registry (using `package-lock.json`, `pnpm-lock.yaml`, `yarn.lock`, or a pinned version in `package.json`), analyzes the unpacked sources, and cleans up temp files per package.

```bash
# Air-gapped or CI: require a full local install
npx @vinean/dependency-analyzer --no-ghost

# Skip registry metadata but still allow tarball fetch for missing packages
npx @vinean/dependency-analyzer --no-registry
```

## 📊 Key Features

### 1. Replaceability Score (0-100)

`@vinean/dependency-analyzer` analyzes your codebase to estimate how difficult it would be to remove a dependency and replace it with your own implementation.

- **Easy (0-30)**: Minimal logic, easy to replace or implement yourself.
- **Medium (31-70)**: Moderate complexity and coupling.
- **Hard (71-100)**: Deeply integrated, native code, or massive API surface.

The score is derived from 5 critical metrics:

- **Native Presence**: Detects C++/CGO/Unsafe code.
- **Code Volume**: Measures physical size and SLOC.
- **API Surface**: Evaluates the breadth and complexity of the interface.
- **Entanglement**: Tracks dependency chains and OS-level integrations.
- **Logic Complexity**: Proxies cognitive load and concurrency features.

### 2. Maintenance & Health

Identify abandoned or deprecated packages before they become a liability.

- **Maintenance Status**: Track if a package is active (Yes), stale (Unlikely), or deprecated (No).
- **Update Recency**: See exactly how long ago the last version was released.
- **Popularity**: Weekly download counts (NPM) provide context on package trust.

### 3. Ecosystem Specifics

- **React Native**: Detects native module usage and "New Architecture" (TurboModule/Fabric) support.
- **Go Support**: Deep analysis of module internals and proxy metadata.

## 📑 Interactive Dashboard

The tool generates a `dependency-report.html` interactive dashboard in your project directory:

- **Sortable Metrics**: Rank dependencies by complexity, downloads, or update age.
- **Export Capabilities**: Download the full analysis as a JSON for CI/CD or internal tools.
- **Offline First**: The report is self-contained and can be viewed without an internet connection.

---

[View full documentation on GitHub](https://github.com/hakanolgun/dependency-analyzer)
