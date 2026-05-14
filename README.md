# @hakanolgun/dependency-analyzer

A CLI for analyzing the **Replaceability**, **Maintenance**, and **Health** of **npm** dependencies in JavaScript and TypeScript projects.

## Usage

Run it directly via `npx` in your project root directory (must contain `package.json`):

```bash
npx @hakanolgun/dependency-analyzer
```

Or install it globally:

```bash
npm install -g @hakanolgun/dependency-analyzer
@hakanolgun/dependency-analyzer --project ./my-cool-project
```

## Options

- `--project <path>`: Path to the project root (default: current directory).
- `--open=false`: Disable auto-opening the generated HTML report.
- `--json`: Print the raw analysis summary (JSON) to stdout.
- `--dev`: Include `devDependencies` in the scan. By default, only production `dependencies` are analyzed.
- `--no-ghost`: Do not fetch package tarballs from the registry when `node_modules` is missing. Analysis uses only what is installed locally.
- `--no-registry`: Skip npm registry metadata (weekly downloads, maintenance heuristics, React Native directory). Does not disable tarball fetch for code analysis; use `--no-ghost` for that.

### NPM: local install first, registry fallback

By default, dependencies are read from `node_modules` when present. If a direct dependency is missing on disk, the tool downloads its **exact** version from the registry (using `package-lock.json`, `pnpm-lock.yaml`, `yarn.lock`, or a pinned version in `package.json`), analyzes the unpacked sources, and cleans up temp files per package.

## Key features

### 1. Replaceability cost (0-100)

Analyzes your codebase to estimate how difficult it would be to remove a dependency and replace it with your own implementation.

- **Easy (0-30)**: Minimal logic, easy to replace or implement yourself.
- **Medium (31-70)**: Moderate complexity and coupling.
- **Hard (71-100)**: Deeply integrated, native code, or massive API surface.

The score is derived from five metrics: native presence, code volume, API surface, entanglement, and logic complexity.

### 2. Maintenance and health

Identify abandoned or deprecated packages before they become a liability.

- **Maintenance status**: Track if a package is active (Yes), stale (Unlikely), or deprecated (No).
- **Update recency**: See exactly how long ago the last version was released.
- **Popularity**: Weekly download counts provide context on package trust.

### 3. React Native

Detects native module usage and "New Architecture" (TurboModule/Fabric) support when applicable.

## 📑 Interactive dashboard

The tool generates a `dependency-report.html` interactive dashboard in your project directory:

- **Sortable metrics**: Rank dependencies by complexity, downloads, or update age.
- **Export capabilities**: Download the full analysis as JSON for CI/CD or internal tools.
- **Offline first**: The report is self-contained and can be viewed without an internet connection.

---

## Key Pillars of Replaceability

Dependency Analyzer evaluates every dependency across five critical metrics to generate a normalized **Replaceability Cost (0-100)**. A higher score indicates a dependency that is more "locked-in" and costly to replace.

### 1. Native Presence (40%)

Detects native code signals (C++, `node-gyp`). Native dependencies often require specific build environments and are significantly harder to port or replace with pure-logic alternatives.

### 2. Code Volume (10%)

Analyzes the physical size and source lines of code (SLOC). While not a direct measure of complexity, massive packages represent a larger "surface area" of logic that your project might be relying on.

### 3. API Surface (10%)

Measures the breadth of the public interface: exports, classes, and methods. **Structural penalty**: higher nesting levels (max brace depth) increase the score.

### 4. Entanglement (15%)

Analyzes dependency chains.

- Tracks direct and peer dependencies.
- Heuristically estimates **transitive depth**.
- Detects "shell leaks" (imports of `child_process`, etc.) that suggest deep OS-level integration.

### 5. Logic Complexity (25%)

A proxy for cognitive complexity: decision point density (if/else, switch, catch). **Confidence modifier**: high test-file counts slightly reduce this score, as well-tested code is easier to refactor or replace.

## License

Distributed under the GNU AFFERO GENERAL PUBLIC LICENSE v3.0. See `LICENSE.md` for more information.
