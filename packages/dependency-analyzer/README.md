# 🛡️ @vinean/dependency-analyzer

A high-performance CLI for analyzing the **Replaceability** of your project's dependencies. Supports both **NPM** and **Go Modules**.

## 🚀 Usage

Run it directly via `npx`:

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
- `--open=false`: Disable auto-opening the generated HTML report.
- `--json`: Print the raw analysis summary (JSON) to stdout.

## 📊 What is Replaceability?

`@vinean/dependency-analyzer` deep-dives into your project's `node_modules` or Go proxy source code to calculate a **Replaceability Score (0-100)** based on:

1. **Native Presence**: Detects C++/CGO/Unsafe code.
2. **Code Volume**: Measures physical size and SLOC.
3. **API Surface**: Evaluates the breadth and complexity of the public interface.
4. **Entanglement**: Tracks dependency chains and OS-level integrations.
5. **Logic Complexity**: Proxies cognitive load and concurrency features.

## 📑 Output

Generates a `dep-report.html` interactive dashboard in your project directory, allowing you to explore the metrics, check maintenance status, and export results.

---

[View full documentation on GitHub](https://github.com/hakanolgun/dependency-analyzer)
