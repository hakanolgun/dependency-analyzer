# 🛡️ Dependency Analyzer

**Dependency-Analyzer** is a powerful dependency analysis tool designed to evaluate the "Replaceability" of your project's dependencies. It doesn't just list your packages; it deep-dives into their source code and metadata to calculate exactly how much effort would be required to replace them.

Available as both a **high-performance Go CLI** and a **modern React Web Interface**.

---

## 🚀 Key Pillars of Replaceability

Dep-Scan evaluates every dependency across five critical metrics to generate a normalized **Replaceability Score (0-100)**. A higher score indicates a dependency that is more "locked-in" and costly to replace.

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

| Ecosystem         | Detected File  | Analysis Level                               |
| :---------------- | :------------- | :------------------------------------------- |
| **NPM / Node.js** | `package.json` | Deep `node_modules` scan + Registry metadata |
| **Go Modules**    | `go.mod`       | Proxy zip download + Source code parsing     |

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

---

## 🖥️ Web Interface

Dep-Scan includes a premium dashboard for exploring analysis results visually.

- **Interactive Sorting**: Sort by download count, last update, or replaceability score.
- **Maintained Status**: Smart heuristics (`yes`, `unlikely`, `no`) based on recent update history.
- **Ecosystem Switching**: Toggle between JavaScript and Go results.
- **JSON Export**: Export the full analysis report for further processing.

To run the dashboard locally:

```bash
npm install
npm run dev
```

---

## 📂 Project Structure

- `cli-go/`: The core analysis engine implemented in Go.
- `packages/dependency-analyzer/`: Node.js wrapper for the CLI distribution.
- `src/`: React + Vite frontend source code.
- `src/lib/`: Unified scoring logic (mirrors Go implementation for frontend-only scans).

---

## 📜 License

Distributed under the GNU AFFERO GENERAL PUBLIC LICENSE v3.0. See `LICENSE.md` for more information.

---
