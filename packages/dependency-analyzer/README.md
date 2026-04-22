# 🛡️ @hakanolgun/dependency-analyzer

A high-performance CLI for analyzing the **Replaceability**, **Maintenance**, and **Health** of **npm** dependencies in JavaScript and TypeScript projects.

## 🚀 Usage

Run it directly via `npx` in your project root directory (must contain `package.json`):

```bash
npx @hakanolgun/dependency-analyzer
```

Or install it globally:

```bash
npm install -g @hakanolgun/dependency-analyzer
@hakanolgun/dependency-analyzer --project ./my-cool-project
```

## ⚙️ Options

- `--project <path>`: Path to the project root (default: current directory).
- `--ecosystem`: Optional; use `npm` or leave empty. The `go` value is no longer supported.
- `--open=false`: Disable auto-opening the generated HTML report.
- `--json`: Print the raw analysis summary (JSON) to stdout.
- `--no-ghost`: Do not fetch package tarballs from the registry when `node_modules` is missing. Analysis uses only what is installed locally.
- `--no-registry`: Skip npm registry metadata (weekly downloads, maintenance heuristics, React Native directory). Does not disable tarball fetch for code analysis; use `--no-ghost` for that.

### NPM: local install first, registry fallback

By default, dependencies are read from `node_modules` when present. If a direct dependency is missing on disk, the tool downloads its **exact** version from the registry (using `package-lock.json`, `pnpm-lock.yaml`, `yarn.lock`, or a pinned version in `package.json`), analyzes the unpacked sources, and cleans up temp files per package.

```bash
# Air-gapped or CI: require a full local install
npx @hakanolgun/dependency-analyzer --no-ghost

# Skip registry metadata but still allow tarball fetch for missing packages
npx @hakanolgun/dependency-analyzer --no-registry
```

## 📊 Key features

### 1. Replaceability cost (0-100)

`@hakanolgun/dependency-analyzer` analyzes your codebase to estimate how difficult it would be to remove a dependency and replace it with your own implementation.

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

[View full documentation on GitHub](https://github.com/hakanolgun/dependency-analyzer)
