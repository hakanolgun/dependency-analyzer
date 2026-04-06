# 🛡️ @vinean/dependency-analyzer

A high-performance CLI for analyzing the **Replaceability**, **Maintenance**, and **Health** of your project's dependencies. Supports both **NPM** and **Go Modules**.

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

## 📊 Key Features

### 1. Replaceability Score (0-100)

`@vinean/dependency-analyzer` deep-dives into your project's source code to calculate how difficult it would be to replace a dependency.

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

The tool generates a `dep-report.html` interactive dashboard in your project directory:

- **Sortable Metrics**: Rank dependencies by complexity, downloads, or update age.
- **Export Capabilities**: Download the full analysis as a JSON for CI/CD or internal tools.
- **Offline First**: The report is self-contained and can be viewed without an internet connection.

---

[View full documentation on GitHub](https://github.com/hakanolgun/dependency-analyzer)
